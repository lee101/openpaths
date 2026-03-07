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

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Landing />} />
          <Route path="models" element={<Models />} />
          <Route path="providers" element={<Providers />} />
          <Route path="playground" element={<Playground />} />
          <Route path="account" element={<Account />} />
          <Route path="blog" element={<Blog />} />
          <Route path="blog/:slug" element={<BlogPost />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
