import React, { useEffect, useMemo, useState } from 'react';
import {
  Folder as FolderIcon, FolderPlus, Plus, Pin, PinOff, Archive, Trash2, Tag as TagIcon,
  ChevronDown, ChevronRight, Search, X, MoreHorizontal, FolderInput, Settings2, Check, ArchiveRestore,
} from 'lucide-react';
import * as conv from '../lib/conversations';
import type { Conversation, Folder, ConvFilter } from '../lib/conversations';

interface Props {
  activeId: string | null;
  onSelect: (c: Conversation) => void;
  onNewChat: (folderId: string | null) => void;
}

type View = { kind: 'all' } | { kind: 'unfiled' } | { kind: 'archived' } | { kind: 'folder'; id: string };

export function ChatSidebar({ activeId, onSelect, onNewChat }: Props) {
  const [, force] = useState(0);
  const [view, setView] = useState<View>({ kind: 'all' });
  const [search, setSearch] = useState('');
  const [tag, setTag] = useState<string>('');
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [selectMode, setSelectMode] = useState(false);
  const [selected, setSelected] = useState(() => new Set<string>());
  const [menuFor, setMenuFor] = useState<string | null>(null);
  const [editingFolder, setEditingFolder] = useState<Folder | null>(null);

  useEffect(() => conv.subscribe(() => force(n => n + 1)), []);

  const folders = conv.listFolders();
  const tags = conv.allTags();

  const items = useMemo<Conversation[]>(() => {
    const f: ConvFilter = { search, tag: tag || undefined };
    if (view.kind === 'archived') f.archived = true;
    else if (view.kind === 'unfiled') f.folderId = null;
    else if (view.kind === 'folder') f.folderId = view.id;
    return conv.filterConversations(f);
  }, [view, search, tag, folders.length]);

  function toggleSel(id: string) {
    setSelected(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  }

  function clearSelect() {
    setSelected(new Set<string>());
    setSelectMode(false);
  }

  const ids: string[] = Array.from(selected);

  return (
    <div className="flex flex-col h-full w-64 shrink-0 border-r border-white/10 bg-black/30 text-sm">
      {/* Header */}
      <div className="flex items-center gap-1 px-2 py-2 border-b border-white/10">
        <button
          onClick={() => onNewChat(view.kind === 'folder' ? view.id : null)}
          className="flex-1 flex items-center justify-center gap-1.5 py-1.5 rounded-md bg-white/10 hover:bg-white/15 text-white/90 text-xs font-medium"
        >
          <Plus className="w-3.5 h-3.5" /> New chat
        </button>
        <button
          title="New folder"
          onClick={() => { const f = conv.createFolder('New folder'); setEditingFolder(f); }}
          className="p-1.5 rounded-md hover:bg-white/10 text-white/70"
        >
          <FolderPlus className="w-4 h-4" />
        </button>
        <button
          title="Select"
          onClick={() => (selectMode ? clearSelect() : setSelectMode(true))}
          className={`p-1.5 rounded-md hover:bg-white/10 ${selectMode ? 'text-emerald-400' : 'text-white/70'}`}
        >
          <Check className="w-4 h-4" />
        </button>
      </div>

      {/* Search */}
      <div className="px-2 py-2 border-b border-white/10 space-y-2">
        <div className="relative">
          <Search className="w-3.5 h-3.5 absolute left-2 top-1/2 -translate-y-1/2 text-white/40" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search chats"
            className="w-full pl-7 pr-2 py-1.5 rounded-md bg-white/5 border border-white/10 text-xs text-white/90 placeholder-white/40 focus:outline-none focus:border-white/25"
          />
        </div>
        {tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {tags.map(t => (
              <button
                key={t}
                onClick={() => setTag(tag === t ? '' : t)}
                className={`px-1.5 py-0.5 rounded text-[10px] flex items-center gap-1 ${tag === t ? 'bg-sky-500/30 text-sky-200' : 'bg-white/5 text-white/50 hover:bg-white/10'}`}
              >
                <TagIcon className="w-2.5 h-2.5" />{t}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Views + folders */}
      <div className="flex-1 overflow-y-auto px-1 py-1">
        <NavRow label="All chats" active={view.kind === 'all'} onClick={() => setView({ kind: 'all' })} icon={<FolderIcon className="w-3.5 h-3.5" />} />
        <NavRow label="Unfiled" active={view.kind === 'unfiled'} count={conv.folderCount(null)} onClick={() => setView({ kind: 'unfiled' })} icon={<FolderIcon className="w-3.5 h-3.5 opacity-50" />} />
        <NavRow label="Archived" active={view.kind === 'archived'} onClick={() => setView({ kind: 'archived' })} icon={<Archive className="w-3.5 h-3.5" />} />

        <div className="mt-2 mb-1 px-2 text-[10px] uppercase tracking-wide text-white/30">Folders</div>
        {folders.map(f => {
          const isOpen = !collapsed[f.id];
          const inView = view.kind === 'folder' && view.id === f.id;
          const folderItems = isOpen ? conv.filterConversations({ folderId: f.id, search, tag: tag || undefined }) : [];
          return (
            <div key={f.id}>
              <div className={`group flex items-center gap-1 px-1.5 py-1 rounded-md cursor-pointer ${inView ? 'bg-white/10' : 'hover:bg-white/5'}`}>
                <button onClick={() => setCollapsed(c => ({ ...c, [f.id]: isOpen }))} className="text-white/40 hover:text-white/70">
                  {isOpen ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
                </button>
                <button onClick={() => setView({ kind: 'folder', id: f.id })} className="flex-1 flex items-center gap-1.5 min-w-0 text-left">
                  <FolderIcon className="w-3.5 h-3.5 shrink-0" style={f.color ? { color: f.color } : undefined} />
                  <span className="truncate text-white/85 text-xs">{f.name}</span>
                  <span className="text-[10px] text-white/30">{conv.folderCount(f.id)}</span>
                </button>
                <button onClick={() => setEditingFolder(f)} className="opacity-0 group-hover:opacity-100 p-0.5 text-white/40 hover:text-white/80">
                  <Settings2 className="w-3.5 h-3.5" />
                </button>
              </div>
              {isOpen && folderItems.map(c => (
                <ConvRow key={c.id} c={c} depth={1} active={c.id === activeId} selectMode={selectMode} selected={selected.has(c.id)}
                  folders={folders} menuOpen={menuFor === c.id}
                  onSelect={() => (selectMode ? toggleSel(c.id) : onSelect(c))}
                  onMenu={() => setMenuFor(menuFor === c.id ? null : c.id)} />
              ))}
            </div>
          );
        })}
      </div>

      {/* Conversation list for non-folder views */}
      {view.kind !== 'folder' && (
        <div className="border-t border-white/10 flex-1 overflow-y-auto px-1 py-1 min-h-[40%]">
          {items.length === 0 && <div className="px-3 py-6 text-center text-xs text-white/30">No chats</div>}
          {items.map((c: Conversation) => (
            <ConvRow key={c.id} c={c} depth={0} active={c.id === activeId} selectMode={selectMode} selected={selected.has(c.id)}
              folders={folders} menuOpen={menuFor === c.id}
              onSelect={() => (selectMode ? toggleSel(c.id) : onSelect(c))}
              onMenu={() => setMenuFor(menuFor === c.id ? null : c.id)} />
          ))}
        </div>
      )}

      {/* Bulk action bar */}
      {selectMode && (
        <div className="border-t border-white/10 p-2 bg-black/40">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-white/60">{selected.size} selected</span>
            <button onClick={clearSelect} className="text-white/40 hover:text-white"><X className="w-3.5 h-3.5" /></button>
          </div>
          <div className="grid grid-cols-3 gap-1">
            <BulkBtn icon={<Archive className="w-3.5 h-3.5" />} label="Archive" disabled={!ids.length} onClick={() => { conv.bulkArchive(ids, view.kind !== 'archived'); clearSelect(); }} />
            <BulkBtn icon={<FolderInput className="w-3.5 h-3.5" />} label="Move" disabled={!ids.length} onClick={() => {
              const name = folders.length ? prompt('Move to folder name (blank = unfiled):', '') : prompt('No folders yet. Name a new folder:', 'New folder');
              if (name === null) return;
              let fid: string | null = null;
              const t = name.trim();
              if (t) { const ex = folders.find(f => f.name.toLowerCase() === t.toLowerCase()); fid = (ex || conv.createFolder(t)).id; }
              conv.bulkMove(ids, fid); clearSelect();
            }} />
            <BulkBtn icon={<Trash2 className="w-3.5 h-3.5" />} label="Delete" danger disabled={!ids.length} onClick={() => { if (confirm(`Delete ${ids.length} chat(s)?`)) { conv.bulkDelete(ids); clearSelect(); } }} />
          </div>
        </div>
      )}

      {editingFolder && <FolderEditor folder={editingFolder} onClose={() => setEditingFolder(null)} />}
      {menuFor && <ConvMenu id={menuFor} folders={folders} archived={view.kind === 'archived'} onClose={() => setMenuFor(null)} />}
    </div>
  );
}

function NavRow({ label, active, count, onClick, icon }: { label: string; active: boolean; count?: number; onClick: () => void; icon: React.ReactNode }) {
  return (
    <button onClick={onClick} className={`w-full flex items-center gap-2 px-2 py-1.5 rounded-md text-xs ${active ? 'bg-white/10 text-white' : 'text-white/70 hover:bg-white/5'}`}>
      {icon}<span className="flex-1 text-left">{label}</span>
      {count != null && count > 0 && <span className="text-[10px] text-white/30">{count}</span>}
    </button>
  );
}

interface ConvRowProps {
  c: Conversation; depth: number; active: boolean; selectMode: boolean; selected: boolean;
  folders: Folder[]; menuOpen: boolean; onSelect: () => void; onMenu: () => void;
}

const ConvRow: React.FC<ConvRowProps> = ({ c, depth, active, selectMode, selected, onSelect, onMenu }) => {
  return (
    <div
      onClick={onSelect}
      className={`group flex items-center gap-1.5 pr-1 py-1 rounded-md cursor-pointer ${active ? 'bg-sky-500/15 ring-1 ring-sky-500/30' : 'hover:bg-white/5'}`}
      style={{ paddingLeft: 8 + depth * 14 }}
    >
      {selectMode && (
        <span className={`w-3.5 h-3.5 rounded border shrink-0 flex items-center justify-center ${selected ? 'bg-emerald-500 border-emerald-500' : 'border-white/30'}`}>
          {selected && <Check className="w-2.5 h-2.5 text-black" />}
        </span>
      )}
      {c.pinned && <Pin className="w-3 h-3 text-amber-400 shrink-0" />}
      <div className="flex-1 min-w-0">
        <div className="truncate text-xs text-white/85">{c.title}</div>
        {c.tags.length > 0 && (
          <div className="flex gap-1 mt-0.5">
            {c.tags.slice(0, 3).map(t => <span key={t} className="px-1 rounded bg-white/5 text-[9px] text-white/40">{t}</span>)}
          </div>
        )}
      </div>
      {!selectMode && (
        <button onClick={e => { e.stopPropagation(); onMenu(); }} className="opacity-0 group-hover:opacity-100 p-0.5 text-white/40 hover:text-white">
          <MoreHorizontal className="w-3.5 h-3.5" />
        </button>
      )}
    </div>
  );
}

function ConvMenu({ id, folders, archived, onClose }: { id: string; folders: Folder[]; archived: boolean; onClose: () => void }) {
  const c = conv.getConversation(id);
  if (!c) return null;
  const act = (fn: () => void) => { fn(); onClose(); };
  return (
    <>
      <div className="fixed inset-0 z-40" onClick={onClose} />
      <div className="fixed z-50 bottom-16 left-2 w-60 rounded-lg border border-white/15 bg-zinc-900 shadow-xl p-1 text-xs">
        <MenuItem icon={c.pinned ? <PinOff className="w-3.5 h-3.5" /> : <Pin className="w-3.5 h-3.5" />} label={c.pinned ? 'Unpin' : 'Pin'} onClick={() => act(() => conv.togglePinned(id))} />
        <MenuItem icon={<Pin className="w-3.5 h-3.5 opacity-0" />} label="Rename" onClick={() => act(() => { const t = prompt('Rename chat:', c.title); if (t) conv.renameConversation(id, t); })} />
        <MenuItem icon={<TagIcon className="w-3.5 h-3.5" />} label="Edit tags" onClick={() => act(() => { const t = prompt('Tags (comma separated):', c.tags.join(', ')); if (t !== null) conv.setTags(id, t.split(',')); })} />
        <div className="px-2 py-1 text-[10px] uppercase text-white/30">Move to</div>
        <MenuItem icon={<FolderIcon className="w-3.5 h-3.5 opacity-50" />} label="Unfiled" onClick={() => act(() => conv.moveToFolder(id, null))} />
        {folders.map(f => <MenuItem key={f.id} icon={<FolderIcon className="w-3.5 h-3.5" />} label={f.name} onClick={() => act(() => conv.moveToFolder(id, f.id))} />)}
        <MenuItem icon={<FolderPlus className="w-3.5 h-3.5" />} label="New folder…" onClick={() => act(() => { const n = prompt('Folder name:'); if (n) conv.moveToFolder(id, conv.createFolder(n).id); })} />
        <div className="my-1 border-t border-white/10" />
        <MenuItem icon={archived ? <ArchiveRestore className="w-3.5 h-3.5" /> : <Archive className="w-3.5 h-3.5" />} label={archived ? 'Unarchive' : 'Archive'} onClick={() => act(() => conv.archiveConversation(id, !archived))} />
        <MenuItem icon={<Trash2 className="w-3.5 h-3.5" />} label="Delete" danger onClick={() => act(() => { if (confirm('Delete this chat?')) conv.deleteConversation(id); })} />
      </div>
    </>
  );
}

interface MenuItemProps { icon: React.ReactNode; label: string; onClick: () => void; danger?: boolean }

const MenuItem: React.FC<MenuItemProps> = ({ icon, label, onClick, danger }) => {
  return (
    <button onClick={onClick} className={`w-full flex items-center gap-2 px-2 py-1.5 rounded text-left hover:bg-white/10 ${danger ? 'text-red-400' : 'text-white/85'}`}>
      {icon}<span className="truncate">{label}</span>
    </button>
  );
}

function BulkBtn({ icon, label, onClick, disabled, danger }: { icon: React.ReactNode; label: string; onClick: () => void; disabled?: boolean; danger?: boolean }) {
  return (
    <button disabled={disabled} onClick={onClick} className={`flex flex-col items-center gap-0.5 py-1.5 rounded text-[10px] disabled:opacity-30 ${danger ? 'text-red-400 hover:bg-red-500/10' : 'text-white/70 hover:bg-white/10'}`}>
      {icon}{label}
    </button>
  );
}

function FolderEditor({ folder, onClose }: { folder: Folder; onClose: () => void }) {
  const [name, setName] = useState(folder.name);
  const [systemPrompt, setSystemPrompt] = useState(folder.systemPrompt || '');
  const [dynamicContext, setDynamicContext] = useState(folder.dynamicContext || '');
  const [defaultModel, setDefaultModel] = useState(folder.defaultModel || '');

  function save() {
    conv.updateFolder(folder.id, { name: name.trim() || folder.name, systemPrompt, dynamicContext, defaultModel: defaultModel.trim() || undefined });
    onClose();
  }

  return (
    <>
      <div className="fixed inset-0 z-50 bg-black/60" onClick={onClose} />
      <div className="fixed z-50 left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[min(92vw,440px)] rounded-xl border border-white/15 bg-zinc-900 p-4 shadow-2xl">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-white flex items-center gap-2"><FolderIcon className="w-4 h-4" /> Project folder</h3>
          <button onClick={onClose} className="text-white/40 hover:text-white"><X className="w-4 h-4" /></button>
        </div>
        <div className="space-y-3 text-xs">
          <Field label="Name"><input value={name} onChange={e => setName(e.target.value)} className={inputCls} /></Field>
          <Field label="Default model (optional)" hint="Model id applied when starting a chat here">
            <input value={defaultModel} onChange={e => setDefaultModel(e.target.value)} placeholder="e.g. claude or auto" className={inputCls} />
          </Field>
          <Field label="System prompt (optional)" hint="Sent as system message for chats in this folder">
            <textarea value={systemPrompt} onChange={e => setSystemPrompt(e.target.value)} rows={3} className={inputCls} />
          </Field>
          <Field label="Dynamic context (optional)" hint="Always prepended — project facts, style, constraints">
            <textarea value={dynamicContext} onChange={e => setDynamicContext(e.target.value)} rows={3} className={inputCls} />
          </Field>
        </div>
        <div className="flex justify-between gap-2 mt-4">
          <button onClick={() => { if (confirm('Delete folder? Chats move to Unfiled.')) { conv.deleteFolder(folder.id); onClose(); } }} className="px-3 py-1.5 rounded-md text-xs text-red-400 hover:bg-red-500/10">Delete folder</button>
          <div className="flex gap-2">
            <button onClick={onClose} className="px-3 py-1.5 rounded-md text-xs text-white/70 hover:bg-white/10">Cancel</button>
            <button onClick={save} className="px-3 py-1.5 rounded-md text-xs bg-sky-500 hover:bg-sky-400 text-white font-medium">Save</button>
          </div>
        </div>
      </div>
    </>
  );
}

const inputCls = 'w-full px-2 py-1.5 rounded-md bg-white/5 border border-white/10 text-white/90 placeholder-white/30 focus:outline-none focus:border-white/25';

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <div className="text-white/60 mb-1">{label}</div>
      {children}
      {hint && <div className="text-[10px] text-white/30 mt-0.5">{hint}</div>}
    </label>
  );
}
