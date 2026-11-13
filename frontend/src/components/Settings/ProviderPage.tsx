import { useState } from 'react';
import { getStore, useStore } from '../../store';
import { ModelsBox } from './ModelsBox';
import { SettingsLayout } from './SettingsLayout';
import { TextField } from './TextField';

/** Edit a provider (id) or add one (id null). */
export function ProviderPage({ id }: { id: string | null }) {
  const p = useStore((s) => s.providers.find((x) => x.id === id));
  const { saveProvider, removeProvider, toast, confirm } = getStore();
  const [name, setName] = useState(p?.name || '');
  const [baseUrl, setBaseUrl] = useState(p?.baseUrl || '');
  const [key, setKey] = useState('');

  const save = async () => {
    const n = name.trim();
    const url = baseUrl.trim().replace(/\/+$/, '');
    if (!n || !url) return toast('Add a name and a base URL first');
    if (!/^https?:\/\//.test(url)) return toast('The base URL needs to start with http:// or https://');
    await saveProvider(p ? p.id : null, { name: n, baseUrl: url, ...(key ? { apiKey: key } : {}) });
    setBaseUrl(url);
    setKey('');
  };
  const remove = () =>
    p &&
    confirm({
      title: `Remove ${p.name}?`,
      text: 'Chats that used it stay, but you’ll need to pick another model to continue them.',
      okLabel: 'Remove',
      ok: () => removeProvider(p),
    });

  const actions = (
    <>
      <button className="btn primary" onClick={save}>
        {p ? 'Save changes' : 'Add provider'}
      </button>
      {p && (
        <button className="btn danger push-right" onClick={remove}>
          Remove provider
        </button>
      )}
    </>
  );

  return (
    <SettingsLayout title={p ? p.name : 'Add a provider'} actions={actions}>
      <TextField id="pvName" label="Name" sub="Shown in the model picker." value={name} onChange={setName} placeholder="Ollama" />
      <TextField
        id="pvUrl"
        label="Base URL"
        sub="The OpenAI-compatible root. Models load from {base}/models and chats go to {base}/chat/completions."
        value={baseUrl}
        onChange={setBaseUrl}
        placeholder="http://localhost:11434/v1"
        spellCheck={false}
      />
      <TextField
        id="pvKey"
        label="API key"
        sub={p?.hasKey ? 'A key is saved. Leave blank to keep it.' : 'Optional for local servers. Stored only in the backend database.'}
        type="password"
        value={key}
        onChange={setKey}
        placeholder={p?.hasKey ? '••••••••••••' : 'sk-…'}
        autoComplete="off"
      />
      {p && (
        <div className="field">
          <label>Models</label>
          <ModelsBox providerId={p.id} />
        </div>
      )}
    </SettingsLayout>
  );
}
