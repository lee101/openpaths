import React, { Suspense, lazy } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ScrollToTop } from './components/ScrollToTop';
import { Landing } from './pages/Landing';
import { Models } from './pages/Models';
import { Account } from './pages/Account';
import { Playground } from './pages/Playground';
import { Fusion } from './pages/Fusion';
import { Compound } from './pages/Compound';
import { Blog } from './pages/Blog';
import { BlogPost } from './pages/BlogPost';
import { OpenPathsHarness } from './pages/OpenPathsHarness';
import { Providers } from './pages/Providers';
import { ProviderPage } from './pages/ProviderPage';
import { Docs } from './pages/Docs';
import { Integrations } from './pages/Integrations';
import { Mcp } from './pages/Mcp';
import { WorksWith } from './pages/WorksWith';
import { ProviderDocs } from './pages/ProviderDocs';
import { Pricing } from './pages/Pricing';
import { ModelPage } from './pages/ModelPage';
import { AdminLee } from './pages/AdminLee';
import { Stats } from './pages/Stats';
import { Search } from './pages/Search';
import { ImageTo3D } from './pages/ImageTo3D';
import { TextTo3D } from './pages/TextTo3D';
import { Rig3D } from './pages/Rig3D';
import { Retexture3D } from './pages/Retexture3D';
import { TextToImage } from './pages/TextToImage';
import { ImageEdit } from './pages/ImageEdit';
import { VideoExtension } from './pages/VideoExtension';
import { Tools } from './pages/Tools';
import { Alternatives } from './pages/Alternatives';
import { Evals } from './pages/Evals';
import { ImageEvals } from './pages/ImageEvals';
import { Compare, CompareIndex } from './pages/Compare';
import { ZImageArt } from './pages/ZImageArt';
import { ArtDetail } from './pages/ArtDetail';
import { UsagePrompts, UsageImages } from './pages/UsageSearch';
import { ArtTag } from './pages/ArtTag';
import { NotFound } from './pages/NotFound';
import { Prompts } from './pages/Prompts';
import { PromptDetail } from './pages/PromptDetail';
import { Agents } from './pages/Agents';
import { AgentDetail } from './pages/AgentDetail';
import { OrgJoin } from './pages/OrgJoin';

const Apps = lazy(() => import('./pages/Apps').then(module => ({ default: module.Apps })));
const AppDetail = lazy(() => import('./pages/AppDetail').then(module => ({ default: module.AppDetail })));
const SharedChat = lazy(() => import('./pages/SharedChat').then(module => ({ default: module.SharedChat })));
const Artifacts = lazy(() => import('./pages/Artifacts').then(module => ({ default: module.Artifacts })));
const ArtifactEditor = lazy(() => import('./pages/ArtifactEditor').then(module => ({ default: module.ArtifactEditor })));
const ArtifactDetail = lazy(() => import('./pages/ArtifactDetail').then(module => ({ default: module.ArtifactDetail })));

export default function App() {
  return (
    <BrowserRouter>
      <ScrollToTop />
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
          <Route path="mcp" element={<Mcp />} />
          <Route path="works-with-openpaths" element={<WorksWith />} />
          <Route path=":slug/docs" element={<ProviderDocs />} />
          <Route path="playground" element={<Playground />} />
          <Route path="chat/:slug" element={<Suspense fallback={<RouteLoading />}><SharedChat /></Suspense>} />
          <Route path="agents" element={<Agents />} />
          <Route path="agents/:id" element={<AgentDetail />} />
		  <Route path="orgs/:slug/join" element={<OrgJoin />} />
          <Route path="fusion" element={<Fusion />} />
          <Route path="compound" element={<Compound />} />
          <Route path="tools" element={<Tools />} />
          <Route path="image-to-3d" element={<ImageTo3D />} />
          <Route path="text-to-3d" element={<TextTo3D />} />
          <Route path="rig-3d" element={<Rig3D />} />
          <Route path="retexture-3d" element={<Retexture3D />} />
          <Route path="text-to-image" element={<TextToImage />} />
          <Route path="image-edit" element={<ImageEdit />} />
          <Route path="video-extension" element={<VideoExtension />} />
          <Route path="search" element={<Search />} />
          <Route path="art" element={<ZImageArt />} />
          <Route path="art/i/:slug" element={<ArtDetail />} />
          <Route path="art/tag/:slug" element={<ArtTag />} />
          <Route path="prompts" element={<Prompts scope="all" />} />
          <Route path="prompts/category/:slug" element={<Prompts scope="category" />} />
          <Route path="prompts/type/:slug" element={<Prompts scope="type" />} />
          <Route path="prompts/model/*" element={<Prompts scope="model" />} />
          <Route path="prompts/:slug" element={<PromptDetail />} />
          <Route path="account" element={<Account />} />
          <Route path="account/apikeys" element={<Account />} />
          <Route path="apikeys" element={<Account />} />
          <Route path="usage" element={<Account />} />
          <Route path="usage/prompts" element={<UsagePrompts />} />
          <Route path="usage/images" element={<UsageImages />} />
          <Route path="admin" element={<AdminLee />} />
          <Route path="stats" element={<Stats />} />
          <Route path="apps" element={<Suspense fallback={<RouteLoading />}><Apps /></Suspense>} />
          <Route path="apps/" element={<Suspense fallback={<RouteLoading />}><Apps /></Suspense>} />
          <Route path="apps/:slug" element={<Suspense fallback={<RouteLoading />}><AppDetail /></Suspense>} />
          <Route path="apps/:slug/" element={<Suspense fallback={<RouteLoading />}><AppDetail /></Suspense>} />
          <Route path="artifacts" element={<Suspense fallback={<RouteLoading />}><Artifacts /></Suspense>} />
          <Route path="artifacts/new" element={<Suspense fallback={<RouteLoading />}><ArtifactEditor /></Suspense>} />
          <Route path="artifacts/:id/edit" element={<Suspense fallback={<RouteLoading />}><ArtifactEditor isEdit /></Suspense>} />
          <Route path="artifacts/:id" element={<Suspense fallback={<RouteLoading />}><ArtifactDetail /></Suspense>} />
          <Route path="evals" element={<Evals />} />
          <Route path="image-evals" element={<ImageEvals />} />
          <Route path="compare" element={<CompareIndex />} />
          <Route path="compare/*" element={<Compare />} />
          <Route path="blog" element={<Blog />} />
          <Route path="blog/:slug" element={<BlogPost />} />
          <Route path="op" element={<OpenPathsHarness />} />
          <Route path="alternatives" element={<Alternatives />} />
          <Route path="alternatives/:slug" element={<BlogPost />} />
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

function RouteLoading() {
  return <div className="mx-auto max-w-7xl px-6 py-16 font-mono text-sm text-white/50">Loading...</div>;
}
