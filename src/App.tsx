import React from 'react';
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
          <Route path="account" element={<Account />} />
          <Route path="blog" element={<Blog />} />
          <Route path="blog/:slug" element={<BlogPost />} />
          <Route path="alternatives/:slug" element={<BlogPost />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
