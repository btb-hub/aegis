import { Link } from 'react-router-dom';

type BreadcrumbItem = {
  label: string;
  href?: string;
};

type PageBreadcrumbProps = {
  items: BreadcrumbItem[];
  ariaLabel: string;
};

export function PageBreadcrumb({ items, ariaLabel }: PageBreadcrumbProps) {
  return (
    <nav aria-label={ariaLabel} className="mb-2 text-sm text-zinc-500">
      <ol className="flex flex-wrap items-center gap-1">
        {items.map((item, index) => {
          const isLast = index === items.length - 1;
          return (
            <li key={`${item.label}-${index}`} className="flex items-center gap-1">
              {index > 0 ? <span aria-hidden="true">/</span> : null}
              {item.href && !isLast ? (
                <Link to={item.href} className="hover:text-zinc-700">
                  {item.label}
                </Link>
              ) : (
                <span className={isLast ? 'text-zinc-900' : undefined}>{item.label}</span>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
