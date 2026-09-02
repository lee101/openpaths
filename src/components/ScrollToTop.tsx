import { useEffect } from 'react';
import { useLocation, useNavigationType } from 'react-router-dom';

/**
 * Resets scroll position on route changes. React Router does not do this by
 * default, so navigating via <Link> kept the previous page's scroll offset -
 * e.g. clicking the footer "Docs" link (scrolled to the bottom of the homepage)
 * landed you at the bottom of the new page. We:
 *   - scroll to the top on a normal PUSH/REPLACE navigation with no hash,
 *   - scroll a #hash target into view when present (so /#api keeps working),
 *   - leave POP (back/forward) alone so the browser's own restoration wins.
 */
export function ScrollToTop() {
  const { pathname, hash } = useLocation();
  const navigationType = useNavigationType();

  useEffect(() => {
    if (navigationType === 'POP') return;

    if (hash) {
      const el = document.getElementById(hash.slice(1));
      if (el) {
        el.scrollIntoView({ behavior: 'auto', block: 'start' });
        return;
      }
    }

    window.scrollTo(0, 0);
  }, [pathname, hash, navigationType]);

  return null;
}
