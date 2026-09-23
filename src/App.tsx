import React, { Suspense, lazy } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ScrollToTop } from './components/ScrollToTop';
import { Landing } from './pages/Landing';
const Models = lazy(() => import('./pages/Models').then(module => ({ default: module.Models })));
const Account = lazy(() => import('./pages/Account').then(module => ({ default: module.Account })));
const Playground = lazy(() => import('./pages/Playground').then(module => ({ default: module.Playground })));
const Fusion = lazy(() => import('./pages/Fusion').then(module => ({ default: module.Fusion })));
const Compound = lazy(() => import('./pages/Compound').then(module => ({ default: module.Compound })));
const Blog = lazy(() => import('./pages/Blog').then(module => ({ default: module.Blog })));
const BlogPost = lazy(() => import('./pages/BlogPost').then(module => ({ default: module.BlogPost })));
const OpenPathsHarness = lazy(() => import('./pages/OpenPathsHarness').then(module => ({ default: module.OpenPathsHarness })));
const Providers = lazy(() => import('./pages/Providers').then(module => ({ default: module.Providers })));
const ProviderPage = lazy(() => import('./pages/ProviderPage').then(module => ({ default: module.ProviderPage })));
const Docs = lazy(() => import('./pages/Docs').then(module => ({ default: module.Docs })));
const Integrations = lazy(() => import('./pages/Integrations').then(module => ({ default: module.Integrations })));
const Mcp = lazy(() => import('./pages/Mcp').then(module => ({ default: module.Mcp })));
const WorksWith = lazy(() => import('./pages/WorksWith').then(module => ({ default: module.WorksWith })));
const ProviderDocs = lazy(() => import('./pages/ProviderDocs').then(module => ({ default: module.ProviderDocs })));
const Pricing = lazy(() => import('./pages/Pricing').then(module => ({ default: module.Pricing })));
const ModelPage = lazy(() => import('./pages/ModelPage').then(module => ({ default: module.ModelPage })));
const AdminLee = lazy(() => import('./pages/AdminLee').then(module => ({ default: module.AdminLee })));
const AdminUserUsage = lazy(() => import('./pages/AdminLee').then(module => ({ default: module.AdminUserUsage })));
const Stats = lazy(() => import('./pages/Stats').then(module => ({ default: module.Stats })));
const Search = lazy(() => import('./pages/Search').then(module => ({ default: module.Search })));
const ImageTo3D = lazy(() => import('./pages/ImageTo3D').then(module => ({ default: module.ImageTo3D })));
const TextTo3D = lazy(() => import('./pages/TextTo3D').then(module => ({ default: module.TextTo3D })));
const Rig3D = lazy(() => import('./pages/Rig3D').then(module => ({ default: module.Rig3D })));
const Retexture3D = lazy(() => import('./pages/Retexture3D').then(module => ({ default: module.Retexture3D })));
const TextToImage = lazy(() => import('./pages/TextToImage').then(module => ({ default: module.TextToImage })));
const ImageEdit = lazy(() => import('./pages/ImageEdit').then(module => ({ default: module.ImageEdit })));
const VideoExtension = lazy(() => import('./pages/VideoExtension').then(module => ({ default: module.VideoExtension })));
const ImageToVideo = lazy(() => import('./pages/ImageToVideo').then(module => ({ default: module.ImageToVideo })));
const TextToVideo = lazy(() => import('./pages/TextToVideo').then(module => ({ default: module.TextToVideo })));
const CharacterAnimator = lazy(() => import('./pages/CharacterAnimator').then(module => ({ default: module.CharacterAnimator })));
const MusicGenerator = lazy(() => import('./pages/MusicGenerator').then(module => ({ default: module.MusicGenerator })));
const RemoveVideoBackground = lazy(() => import('./pages/RemoveVideoBackground').then(module => ({ default: module.RemoveVideoBackground })));
const Tools = lazy(() => import('./pages/Tools').then(module => ({ default: module.Tools })));
const LiveVoice = lazy(() => import('./pages/LiveVoice').then(module => ({ default: module.LiveVoice })));
const GoogleTTS = lazy(() => import('./pages/GoogleTTS').then(module => ({ default: module.GoogleTTS })));
const LyriaStudio = lazy(() => import('./pages/LyriaStudio').then(module => ({ default: module.LyriaStudio })));
const Alternatives = lazy(() => import('./pages/Alternatives').then(module => ({ default: module.Alternatives })));
const Evals = lazy(() => import('./pages/Evals').then(module => ({ default: module.Evals })));
const ImageEvals = lazy(() => import('./pages/ImageEvals').then(module => ({ default: module.ImageEvals })));
const Compare = lazy(() => import('./pages/Compare').then(module => ({ default: module.Compare })));
const CompareIndex = lazy(() => import('./pages/Compare').then(module => ({ default: module.CompareIndex })));
const ZImageArt = lazy(() => import('./pages/ZImageArt').then(module => ({ default: module.ZImageArt })));
const ArtDetail = lazy(() => import('./pages/ArtDetail').then(module => ({ default: module.ArtDetail })));
const UsagePrompts = lazy(() => import('./pages/UsageSearch').then(module => ({ default: module.UsagePrompts })));
const UsageImages = lazy(() => import('./pages/UsageSearch').then(module => ({ default: module.UsageImages })));
const ArtTag = lazy(() => import('./pages/ArtTag').then(module => ({ default: module.ArtTag })));
const NotFound = lazy(() => import('./pages/NotFound').then(module => ({ default: module.NotFound })));
const Prompts = lazy(() => import('./pages/Prompts').then(module => ({ default: module.Prompts })));
const PromptDetail = lazy(() => import('./pages/PromptDetail').then(module => ({ default: module.PromptDetail })));
const Agents = lazy(() => import('./pages/Agents').then(module => ({ default: module.Agents })));
const AgentDetail = lazy(() => import('./pages/AgentDetail').then(module => ({ default: module.AgentDetail })));
const Skills = lazy(() => import('./pages/Skills').then(module => ({ default: module.Skills })));
const SkillDetail = lazy(() => import('./pages/SkillDetail').then(module => ({ default: module.SkillDetail })));
const SkillEdit = lazy(() => import('./pages/SkillEdit').then(module => ({ default: module.SkillEdit })));
const OrgJoin = lazy(() => import('./pages/OrgJoin').then(module => ({ default: module.OrgJoin })));
const Byok = lazy(() => import('./pages/Byok').then(module => ({ default: module.Byok })));
const Calculator = lazy(() => import('./pages/Calculator').then(module => ({ default: module.Calculator })));
const Status = lazy(() => import('./pages/Status').then(module => ({ default: module.Status })));
const Teams = lazy(() => import('./pages/Teams').then(module => ({ default: module.Teams })));
const UseCasesIndex = lazy(() => import('./pages/UseCases').then(module => ({ default: module.UseCasesIndex })));
const UseCaseDetail = lazy(() => import('./pages/UseCases').then(module => ({ default: module.UseCaseDetail })));

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
          <Route path="skills" element={<Skills />} />
          <Route path="skills/new" element={<SkillEdit mode="new" />} />
          <Route path="skills/:slug/edit" element={<SkillEdit mode="edit" />} />
          <Route path="skills/:slug" element={<SkillDetail />} />
		  <Route path="orgs/:slug/join" element={<OrgJoin />} />
          <Route path="fusion" element={<Fusion />} />
          <Route path="compound" element={<Compound />} />
          <Route path="tools" element={<Tools />} />
          <Route path="tools/live-voice" element={<LiveVoice />} />
          <Route path="tools/google-tts" element={<GoogleTTS />} />
          <Route path="tools/lyria" element={<LyriaStudio />} />
          <Route path="image-to-3d" element={<ImageTo3D />} />
          <Route path="text-to-video" element={<TextToVideo />} />
          <Route path="image-to-video" element={<ImageToVideo />} />
          <Route path="text-to-3d" element={<TextTo3D />} />
          <Route path="rig-3d" element={<Rig3D />} />
          <Route path="retexture-3d" element={<Retexture3D />} />
          <Route path="text-to-image" element={<TextToImage />} />
          <Route path="image-edit" element={<ImageEdit />} />
          <Route path="video-extension" element={<VideoExtension />} />
          <Route path="character-animator" element={<CharacterAnimator />} />
          <Route path="music-generator" element={<MusicGenerator />} />
          <Route path="remove-video-background" element={<RemoveVideoBackground />} />
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
          <Route path="adminlee" element={<AdminLee />} />
          <Route path="admin/users/:userId/usage" element={<AdminUserUsage />} />
          <Route path="stats" element={<Stats />} />
          <Route path="status" element={<Status />} />
          <Route path="byok" element={<Byok />} />
          <Route path="calculator" element={<Calculator />} />
          <Route path="teams" element={<Teams />} />
          <Route path="use-cases" element={<UseCasesIndex />} />
          <Route path="use-cases/:slug" element={<UseCaseDetail />} />
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
