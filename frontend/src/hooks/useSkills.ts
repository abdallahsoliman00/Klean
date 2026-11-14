import { useCallback, useState } from 'react';
import { api } from '../api';
import type { Skill, SkillInput } from '../types';

export type SkillsState = { status: 'idle' | 'loading' } | { status: 'error'; error: string } | { status: 'ready'; skills: Skill[] };

/** The skill being written or edited. `id` is null for a new one; `source` names the file it was imported from. */
export interface SkillDraft extends Required<SkillInput> {
  id: string | null;
  source?: string;
}

export const NEW_SKILL: SkillDraft = { id: null, name: '', description: '', instructions: '', enabled: true };

/** The saved skills, loaded on first use, and the one open in the editor. Both are kept while the app
    runs, so leaving the page (or closing Settings) loses nothing that was typed. */
export function useSkills() {
  const [state, setState] = useState<SkillsState>({ status: 'idle' });
  const [draft, setDraft] = useState<SkillDraft | null>(null);
  const change = (fn: (skills: Skill[]) => Skill[]) => setState((s) => (s.status === 'ready' ? { ...s, skills: fn(s.skills) } : s));

  const load = useCallback(() => {
    setState({ status: 'loading' });
    api.skills.list().then(
      (skills) => setState({ status: 'ready', skills }),
      (e: Error) => setState({ status: 'error', error: e.message }),
    );
  }, []);

  /** Creates the skill, or updates it when it has an id. */
  const save = async ({ id, source: _, ...input }: SkillDraft) => {
    const saved = id ? await api.skills.update(id, input) : await api.skills.create(input);
    change((skills) => (id ? skills.map((s) => (s.id === id ? saved : s)) : [...skills, saved]));
    return saved;
  };

  /** Switches a skill on or off. The switch moves at once and moves back if the backend refuses. */
  const setEnabled = async (skill: Skill, enabled: boolean) => {
    const put = (next: Skill) => change((skills) => skills.map((s) => (s.id === skill.id ? next : s)));
    put({ ...skill, enabled });
    try {
      put(await api.skills.update(skill.id, { enabled }));
    } catch (e) {
      put(skill);
      throw e;
    }
  };

  const remove = async (id: string) => {
    await api.skills.remove(id);
    change((skills) => skills.filter((s) => s.id !== id));
  };

  return { state, load, save, setEnabled, remove, draft, setDraft };
}
