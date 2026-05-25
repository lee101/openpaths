import React, { Suspense, lazy } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Landing } from './pages/Landing';
import { Models } from './pages/Models';
import { Account } from './pages/Account';
import { Playground } from './pages/Playground';
import { Blog } from './pages/Blog';
import { BlogPost } from './pages/BlogPost';
import { Providers } from './pages/Providers';
import { ProviderPage } from './pages/ProviderPage';
import { Docs } from './pages/Docs';
import { Integrations } from './pages/Integrations';
import { ProviderDocs } from './pages/ProviderDocs';
import { Pricing } from './pages/Pricing';
import { ModelPage } from './pages/ModelPage';
import { AdminLee } from './pages/AdminLee';
import { Stats } from './pages/Stats';
import { Search } from './pages/Search';
import { ImageTo3D } from './pages/ImageTo3D';
import { Alternatives } from './pages/Alternatives';
import { Evals } from './pages/Evals';
import { Compare, CompareIndex } from './pages/Compare';

const Apps = lazy(() => import('./pages/Apps').then(module => ({ default: module.Apps })));
const AppDetail = lazy(() => import('./pages/AppDetail').then(module => ({ default: module.AppDetail })));

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Landing />} />
          <Route path="pricing" element={<Pricing />} />
          <Route path="models" element={<Models />} />
          <Route path="models/:modelId" element={<ModelPage />} />
          <Route path="providers" element={<Providers />} />
          <Route path="providers/:slug" element={<ProviderPage />} />
          <Route path="docs" element={<Docs />} />
          <Route path="integrations" element={<Integrations />} />
          <Route path=":slug/docs" element={<ProviderDocs />} />
          <Route path="playground" element={<Playground />} />
          <Route path="image-to-3d" element={<ImageTo3D />} />
          <Route path="search" element={<Search />} />
          <Route path="account" element={<Account />} />
          <Route path="admin" element={<AdminLee />} />
          <Route path="stats" element={<Stats />} />
          <Route path="apps" element={<Suspense fallback={<RouteLoading />}><Apps /></Suspense>} />
          <Route path="apps/" element={<Suspense fallback={<RouteLoading />}><Apps /></Suspense>} />
          <Route path="apps/:slug" element={<Suspense fallback={<RouteLoading />}><AppDetail /></Suspense>} />
          <Route path="apps/:slug/" element={<Suspense fallback={<RouteLoading />}><AppDetail /></Suspense>} />
          <Route path="evals" element={<Evals />} />
          <Route path="compare" element={<CompareIndex />} />
          <Route path="compare/*" element={<Compare />} />
          <Route path="blog" element={<Blog />} />
          <Route path="blog/:slug" element={<BlogPost />} />
          <Route path="alternatives" element={<Alternatives />} />
          <Route path="alternatives/:slug" element={<BlogPost />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

function RouteLoading() {
  return <div className="mx-auto max-w-7xl px-6 py-16 font-mono text-sm text-white/35">Loading...</div>;
}
