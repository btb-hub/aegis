import type { ReactNode } from 'react';

type PageContentProps = {
  children: ReactNode;
  className?: string;
};

export function PageContent({ children, className = '' }: PageContentProps) {
  return <div className={`min-w-0 space-y-6 ${className}`.trim()}>{children}</div>;
}
