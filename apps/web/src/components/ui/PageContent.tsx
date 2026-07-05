import type { ReactNode } from 'react';

type PageContentProps = {
  children: ReactNode;
  className?: string;
};

export function PageContent({ children, className = '' }: PageContentProps) {
  return <div className={`space-y-6 ${className}`.trim()}>{children}</div>;
}
