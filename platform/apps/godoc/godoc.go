// Package godoc is an app that displays godoc/godoc.org documentation
// for the repository's Go code.
package godoc

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/gddo/doc"
	"github.com/sourcegraph/gddo/gosrc"
	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/sourcegraph/app/router"
	"sourcegraph.com/sourcegraph/sourcegraph/errcode"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/platform"
	"sourcegraph.com/sourcegraph/sourcegraph/platform/apps/godoc/godocsupport"
	"sourcegraph.com/sourcegraph/sourcegraph/platform/pctx"
	"sourcegraph.com/sourcegraph/sourcegraph/util/httputil"
	"sourcegraph.com/sourcegraph/sourcegraph/util/httputil/httpctx"
)

func init() {
	platform.RegisterFrame(platform.RepoFrame{
		ID:      "godoc",
		Title:   "godoc",
		Icon:    "book",
		Handler: http.HandlerFunc(handler),
		Enable:  func(repo *sourcegraph.Repo) bool { return strings.EqualFold(repo.Language, "go") },
	})
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := httpctx.FromRequest(r)
	cl, err := sourcegraph.NewClientFromContext(ctx)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}

	repoRev, exists := pctx.RepoRevSpec(ctx)
	if !exists {
		http.Error(w, "could not parse repository spec from URL", http.StatusBadRequest)
		return
	}

	repo, err := cl.Repos.Get(ctx, &repoRev.RepoSpec)
	if err != nil {
		http.Error(w, err.Error(), errcode.HTTP(err))
		return
	}

	if repoRev.Rev == "" {
		repoRev.Rev = repo.DefaultBranch
	}
	if len(repoRev.CommitID) != 40 {
		commit, err := cl.Repos.GetCommit(ctx, &repoRev)
		if err != nil {
			http.Error(w, "GetCommit: "+err.Error(), http.StatusInternalServerError)
			return
		}
		repoRev.CommitID = string(commit.ID)
	}

	pkg, subpkgs, pdoc, err := build(ctx, repo, repoRev, pathpkg.Clean(r.URL.Path))
	if err != nil {
		http.Error(w, err.Error(), errcode.HTTP(err))
		return
	}

	var bw httputil.ResponseBuffer

	var title string
	if pdoc.Name != "" {
		title = pdoc.Name + " - doc"
	} else {
		title = "godoc"
	}
	bw.Header().Set(platform.HTTPHeaderTitle, title)

	data := &struct {
		Repo        *sourcegraph.Repo
		RepoRevSpec sourcegraph.RepoRevSpec
		Pkg         *doc.Package
		Subpkgs     []*godocsupport.Package
		PDoc        *godocsupport.TDoc
	}{
		Repo:        repo,
		RepoRevSpec: repoRev,
		Pkg:         pkg,
		Subpkgs:     subpkgs,
		PDoc:        pdoc,
	}
	if err := tmpl.Execute(&bw, data); err != nil {
		http.Error(w, err.Error(), errcode.HTTP(err))
		return
	}
	bw.WriteTo(w)
}

var tmpl = template.Must(template.New("godoc").Funcs(godocsupport.TemplateFuncMap).Funcs(funcMap).Parse(tmplHTML))

var funcMap = template.FuncMap{
	"json": func(v interface{}) string {
		b, _ := json.Marshal(v)
		return string(b)
	},
	"urlToRepoGoDoc": func(repo, rev, path string) (*url.URL, error) {
		return router.Rel.URLToOrError(router.RepoAppFrame, "Repo", repo, "Rev", rev, "App", "godoc", "AppPath", "/"+path)
	},
	"pathBase":  pathpkg.Base,
	"hasPrefix": strings.HasPrefix,
}

func build(ctx context.Context, repo *sourcegraph.Repo, repoRev sourcegraph.RepoRevSpec, path string) (*doc.Package, []*godocsupport.Package, *godocsupport.TDoc, error) {
	cl, err := sourcegraph.NewClientFromContext(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	dir, err := getGodocDir(ctx, cl, repo, repoRev, path)
	if err != nil {
		return nil, nil, nil, err
	}

	pkg, err := doc.NewPackage(dir)
	if err != nil {
		return nil, nil, nil, err
	}

	subpkgs := make([]*godocsupport.Package, len(dir.Subdirectories))
	for i, subdir := range dir.Subdirectories {
		subpkgs[i] = &godocsupport.Package{
			Path:     subdir,
			Synopsis: "",
		}
	}

	return pkg, subpkgs, godocsupport.NewTDoc(pkg, dir), nil
}

func getGodocDir(ctx context.Context, cl *sourcegraph.Client, repo *sourcegraph.Repo, repoRevSpec sourcegraph.RepoRevSpec, subdir string) (*gosrc.Directory, error) {
	var importPath string
	if repoRevSpec.URI == "github.com/golang/go" {
		importPath = strings.TrimPrefix(subdir, "src/")
	} else {
		importPath = pathpkg.Join(repoRevSpec.URI, subdir)
	}

	d := &gosrc.Directory{
		ImportPath:  importPath,
		ProjectRoot: repoRevSpec.URI,
		ProjectName: repo.Name,
		ProjectURL:  string(repo.HomepageURL),
		VCS:         "git",
		DeadEndFork: repo.Fork,
		BrowseURL:   router.Rel.URLToRepoTreeEntry(repo.URI, repoRevSpec.CommitID, subdir).String(),
		LineFmt:     "%s#L%d-%[2]d",
	}

	entrySpec := sourcegraph.TreeEntrySpec{
		RepoRev: repoRevSpec,
		Path:    subdir,
	}
	dirEntry, err := cl.RepoTree.Get(ctx, &sourcegraph.RepoTreeGetOp{Entry: entrySpec, Opt: nil})
	if err != nil {
		return nil, err
	}
	for _, entry := range dirEntry.Entries {
		path := pathpkg.Join(subdir, entry.Name)
		switch entry.Type {
		case sourcegraph.FileEntry:
			if filepath.Ext(entry.Name) == ".go" {
				file, err := cl.RepoTree.Get(ctx, &sourcegraph.RepoTreeGetOp{Entry: sourcegraph.TreeEntrySpec{RepoRev: repoRevSpec, Path: path}, Opt: nil})
				if err != nil {
					return nil, err
				}
				d.Files = append(d.Files, &gosrc.File{
					Name:      entry.Name,
					Data:      file.Contents,
					BrowseURL: router.Rel.URLToRepoTreeEntry(repo.URI, repoRevSpec.CommitID, path).String(),
				})
			}
		case sourcegraph.DirEntry:
			d.Subdirectories = append(d.Subdirectories, path)
		}
	}

	return d, nil
}
