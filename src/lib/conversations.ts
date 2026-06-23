// Client-side chat organization layer: project folders, saved conversations,
// pins, tags, archive, bulk ops, and per-folder dynamic context.
// Chats are localStorage-only in this app; this is the explicit "saved" layer
// that sits alongside the per-model scratch history in Playground.

const STORE_KEY = 'op_conversations_v1';

export interface ConvMessage {
  role: 'system' | 'user' | 'assistant';
  content: string;
  createdAt?: number;
  [k: string]: any;
}

export interface Folder {
  id: string;
  name: string;
  color?: string;
  systemPrompt?: string;
  defaultModel?: string;
  dynamicContext?: string;
  createdAt: number;
}

export interface Conversation {
  id: string;
  title: string;
  folderId: string | null;
  tags: string[];
  pinned: boolean;
  archivedAt: number | null;
  model: string;
  systemPrompt?: string;
  messages: ConvMessage[];
  createdAt: number;
  updatedAt: number;
}

interface Store {
  folders: Folder[];
  conversations: Conversation[];
}

type Listener = () => void;
const listeners = new Set<Listener>();

function emptyStore(): Store {
  return { folders: [], conversations: [] };
}

function read(): Store {
  if (typeof window === 'undefined') return emptyStore();
  try {
    const raw = localStorage.getItem(STORE_KEY);
    if (!raw) return emptyStore();
    const parsed = JSON.parse(raw);
    return {
      folders: Array.isArray(parsed?.folders) ? parsed.folders : [],
      conversations: Array.isArray(parsed?.conversations) ? parsed.conversations : [],
    };
  } catch {
    return emptyStore();
  }
}

function write(store: Store) {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(STORE_KEY, JSON.stringify(store));
  } catch {}
  listeners.forEach(l => {
    try { l(); } catch {}
  });
}

export function subscribe(fn: Listener): () => void {
  listeners.add(fn);
  if (typeof window !== 'undefined') {
    // Cross-tab sync.
    window.addEventListener('storage', onStorage);
  }
  return () => {
    listeners.delete(fn);
    if (listeners.size === 0 && typeof window !== 'undefined') {
      window.removeEventListener('storage', onStorage);
    }
  };
}

function onStorage(e: StorageEvent) {
  if (e.key === STORE_KEY) listeners.forEach(l => { try { l(); } catch {} });
}

function uid(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) return crypto.randomUUID();
  return `c_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 9)}`;
}

function now(): number {
  return Date.now();
}

// ---- Reads ----

export function getStore(): Store {
  return read();
}

export function listFolders(): Folder[] {
  return read().folders.slice().sort((a, b) => a.createdAt - b.createdAt);
}

export function getFolder(id: string | null): Folder | undefined {
  if (!id) return undefined;
  return read().folders.find(f => f.id === id);
}

export function listConversations(): Conversation[] {
  return read().conversations.slice().sort(sortConversations);
}

export function getConversation(id: string): Conversation | undefined {
  return read().conversations.find(c => c.id === id);
}

export function allTags(): string[] {
  const set = new Set<string>();
  for (const c of read().conversations) for (const t of c.tags) set.add(t);
  return Array.from(set).sort();
}

function sortConversations(a: Conversation, b: Conversation): number {
  if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
  return b.updatedAt - a.updatedAt;
}

export interface ConvFilter {
  folderId?: string | null;   // undefined = any; null = unfiled
  archived?: boolean;          // default false (active only)
  tag?: string;
  search?: string;
}

export function filterConversations(f: ConvFilter): Conversation[] {
  const search = (f.search || '').trim().toLowerCase();
  return listConversations().filter(c => {
    const isArchived = c.archivedAt != null;
    if ((f.archived ?? false) !== isArchived) return false;
    if (f.folderId !== undefined && c.folderId !== f.folderId) return false;
    if (f.tag && !c.tags.includes(f.tag)) return false;
    if (search) {
      const hay = (c.title + ' ' + c.tags.join(' ') + ' ' + c.model).toLowerCase();
      const inBody = c.messages.some(m => typeof m.content === 'string' && m.content.toLowerCase().includes(search));
      if (!hay.includes(search) && !inBody) return false;
    }
    return true;
  });
}

export function folderCount(folderId: string | null): number {
  return read().conversations.filter(c => c.archivedAt == null && c.folderId === folderId).length;
}

// ---- Folders ----

export function createFolder(name: string, extra?: Partial<Folder>): Folder {
  const store = read();
  const folder: Folder = {
    id: uid(),
    name: name.trim() || 'New folder',
    createdAt: now(),
    ...extra,
  };
  store.folders.push(folder);
  write(store);
  return folder;
}

export function updateFolder(id: string, patch: Partial<Folder>): void {
  const store = read();
  const f = store.folders.find(x => x.id === id);
  if (!f) return;
  Object.assign(f, patch);
  write(store);
}

export function deleteFolder(id: string, opts?: { deleteChats?: boolean }): void {
  const store = read();
  store.folders = store.folders.filter(f => f.id !== id);
  if (opts?.deleteChats) {
    store.conversations = store.conversations.filter(c => c.folderId !== id);
  } else {
    for (const c of store.conversations) if (c.folderId === id) c.folderId = null;
  }
  write(store);
}

// ---- Conversations ----

export interface SaveConvInput {
  id?: string;
  title?: string;
  folderId?: string | null;
  model: string;
  systemPrompt?: string;
  messages: ConvMessage[];
  tags?: string[];
}

export function saveConversation(input: SaveConvInput): Conversation {
  const store = read();
  const ts = now();
  let conv = input.id ? store.conversations.find(c => c.id === input.id) : undefined;
  if (conv) {
    conv.title = input.title ?? conv.title;
    if (input.folderId !== undefined) conv.folderId = input.folderId;
    conv.model = input.model;
    conv.systemPrompt = input.systemPrompt;
    conv.messages = input.messages;
    if (input.tags) conv.tags = input.tags;
    conv.updatedAt = ts;
  } else {
    conv = {
      id: input.id || uid(),
      title: (input.title || deriveTitle(input.messages)).slice(0, 120),
      folderId: input.folderId ?? null,
      tags: input.tags || [],
      pinned: false,
      archivedAt: null,
      model: input.model,
      systemPrompt: input.systemPrompt,
      messages: input.messages,
      createdAt: ts,
      updatedAt: ts,
    };
    store.conversations.push(conv);
  }
  write(store);
  return conv;
}

export function deriveTitle(messages: ConvMessage[]): string {
  const firstUser = messages.find(m => m.role === 'user' && typeof m.content === 'string' && m.content.trim());
  const text = (firstUser?.content || 'New chat').trim().replace(/\s+/g, ' ');
  return text.slice(0, 60) || 'New chat';
}

export function renameConversation(id: string, title: string): void {
  mutateConv(id, c => { c.title = title.trim().slice(0, 120) || c.title; });
}

export function setPinned(id: string, pinned: boolean): void {
  mutateConv(id, c => { c.pinned = pinned; });
}

export function togglePinned(id: string): void {
  mutateConv(id, c => { c.pinned = !c.pinned; });
}

export function moveToFolder(id: string, folderId: string | null): void {
  mutateConv(id, c => { c.folderId = folderId; });
}

export function setTags(id: string, tags: string[]): void {
  mutateConv(id, c => { c.tags = Array.from(new Set(tags.map(t => t.trim()).filter(Boolean))); });
}

export function addTag(id: string, tag: string): void {
  const t = tag.trim();
  if (!t) return;
  mutateConv(id, c => { if (!c.tags.includes(t)) c.tags.push(t); });
}

export function removeTag(id: string, tag: string): void {
  mutateConv(id, c => { c.tags = c.tags.filter(t => t !== tag); });
}

export function archiveConversation(id: string, archived = true): void {
  mutateConv(id, c => { c.archivedAt = archived ? now() : null; });
}

export function deleteConversation(id: string): void {
  const store = read();
  store.conversations = store.conversations.filter(c => c.id !== id);
  write(store);
}

// ---- Bulk ----

export function bulkArchive(ids: string[], archived = true): void {
  const set = new Set(ids);
  const store = read();
  for (const c of store.conversations) if (set.has(c.id)) c.archivedAt = archived ? now() : null;
  write(store);
}

export function bulkDelete(ids: string[]): void {
  const set = new Set(ids);
  const store = read();
  store.conversations = store.conversations.filter(c => !set.has(c.id));
  write(store);
}

export function bulkMove(ids: string[], folderId: string | null): void {
  const set = new Set(ids);
  const store = read();
  for (const c of store.conversations) if (set.has(c.id)) c.folderId = folderId;
  write(store);
}

function mutateConv(id: string, fn: (c: Conversation) => void): void {
  const store = read();
  const c = store.conversations.find(x => x.id === id);
  if (!c) return;
  fn(c);
  c.updatedAt = now();
  write(store);
}

// ---- Dynamic context: compose the effective system prompt for a conversation ----

export function effectiveSystemPrompt(conv: Pick<Conversation, 'folderId' | 'systemPrompt'>, basePrompt: string): string {
  const parts: string[] = [];
  const folder = getFolder(conv.folderId);
  if (folder?.dynamicContext?.trim()) parts.push(folder.dynamicContext.trim());
  if (folder?.systemPrompt?.trim()) parts.push(folder.systemPrompt.trim());
  const own = (conv.systemPrompt ?? basePrompt ?? '').trim();
  if (own) parts.push(own);
  return parts.join('\n\n');
}
