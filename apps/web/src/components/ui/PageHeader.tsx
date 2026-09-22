import type { ReactNode } from 'react';
import { PageBreadcrumb } from './PageBreadcrumb';

type BreadcrumbConfig = {
  ariaLabel: string;
  items: Array<{ label: string; href?: string }>;
};

type PageHeaderProps = {
  title: string;
  subtitle?: string;
  breadcrumb?: BreadcrumbConfig;
  actions?: ReactNode;
};

export function PageHeader({ title, subtitle, breadcrumb, actions }: PageHeaderProps) {
  return (
    <header className="min-w-0 space-y-2">
      {breadcrumb ? (
        <PageBreadcrumb ariaLabel={breadcrumb.ariaLabel} items={breadcrumb.items} />
      ) : null}
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <h1 className="break-words text-3xl font-semibold leading-10 text-zinc-900 [overflow-wrap:anywhere] sm:text-[32px]">
            {title}
          </h1>
          {subtitle ? <p className="mt-1 text-sm leading-[21px] text-zinc-600">{subtitle}</p> : null}
        </div>
        {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
      </div>
    </header>
  );
}
