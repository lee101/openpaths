import { useEffect, type ReactNode } from 'react';

export function NoIndex({ children }: { children: ReactNode }) {
  useEffect(() => {
    const tag = document.createElement('meta');
    tag.setAttribute('name', 'robots');
    tag.setAttribute('content', 'noindex,nofollow');
    document.head.appendChild(tag);
    return () => tag.remove();
  }, []);
  return <>{children}</>;
}
