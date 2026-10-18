import { routeMeta } from '@/config/seo';
import { applyPageMeta } from '@/lib/seo';
import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

/** Sets page metadata for fixed routes. Product pages set their own, so they are skipped here. */
export default function RouteMeta() {
  const { pathname } = useLocation();

  useEffect(() => {
    if (pathname.startsWith('/products/')) return;
    const meta = routeMeta[pathname];
    if (meta) applyPageMeta(meta);
    else applyPageMeta({ title: 'Page not found', noindex: true });
  }, [pathname]);

  return null;
}
