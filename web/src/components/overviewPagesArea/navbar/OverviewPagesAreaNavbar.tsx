import React from 'react'
import { NavLink } from 'react-router-dom'
import { OverviewPagesAreaPage } from '../OverviewPagesArea'

interface Props {
    areaUrl: string
    pages: Pick<OverviewPagesAreaPage<never>, 'title' | 'icon' | 'count' | 'path' | 'exact'>[]
    className?: string
}

const NAV_LINK_CLASS_NAME = 'overview-pages-area-navbar__nav-link nav-link rounded-0 px-3'

/**
 * The navbar for {@link OverviewPagesArea}.
 */
export const OverviewPagesAreaNavbar: React.FunctionComponent<Props> = ({ areaUrl, pages, className = '' }) => (
    <nav className={`overview-pages-area-navbar border-bottom ${className}`}>
        <div className="container">
            <ul className="nav flex-nowrap">
                {pages.map(({ title, icon: Icon, count, path, exact }, i) => (
                    <li key={i} className="overview-pages-area-navbar__nav-item nav-item">
                        <NavLink
                            to={path ? `${areaUrl}${path}` : areaUrl}
                            exact={exact}
                            className={NAV_LINK_CLASS_NAME}
                            activeClassName="overview-pages-area-navbar__nav-link--active"
                            aria-label={title}
                        >
                            {Icon && <Icon className="icon-inline" />} {title}{' '}
                            {count !== undefined && <span className="badge badge-secondary ml-1">{count}</span>}
                        </NavLink>
                    </li>
                ))}
            </ul>
        </div>
    </nav>
)
