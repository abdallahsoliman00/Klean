import { useRef, useState } from 'react';
import { SKILL_COUNT_FROM_SHARE, SKILL_DESCRIPTION_MAX_CHARS } from '../../config';
import type { SkillDraft, useSkills } from '../../hooks/useSkills';
import { SKILL_FIELDS, skillName, skillProblems, type SkillField } from '../../lib/skills';
import { getStore } from '../../store';
import type { Skill } from '../../types';
import { SettingsLayout } from './SettingsLayout';

type Props = Pick<ReturnType<typeof useSkills>, 'save' | 'remove' | 'setDraft'> & { draft: SkillDraft; saved: Skill[] };

/** Writes or edits one skill. Problems show under their field as soon as there is something to say:
    a name that is taken or a description that is too long straight away, an empty field once it has
    been left or Save was pressed. */
export function SkillEditor({ draft, saved, save, remove, setDraft }: Props) {
  const [touched, setTouched] = useState<ReadonlySet<SkillField>>(new Set());
  const [attempted, setAttempted] = useState(false);
  const [saving, setSaving] = useState(false);
  const fields = useRef<Partial<Record<SkillField, HTMLElement | null>>>({});
  const { toast, confirm } = getStore();

  const problems = skillProblems(draft, saved.filter((s) => s.id !== draft.id));
  const shown = (f: SkillField) => (attempted || touched.has(f) || draft[f].trim() ? problems[f] : undefined);
  const set = (patch: Partial<SkillDraft>) => setDraft({ ...draft, ...patch });
  const close = () => setDraft(null);
  const count = draft.description.trim().length;

  /** What every field's control needs: its value handlers, and the wiring to its error text. */
  const control = (f: SkillField, clean: (v: string) => string = (v) => v) => ({
    id: `skill-${f}`,
    ref: (el: HTMLElement | null) => void (fields.current[f] = el),
    value: draft[f],
    'aria-invalid': !!shown(f),
    'aria-describedby': `skill-${f}-help`,
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => set({ [f]: clean(e.target.value) }),
    onBlur: () => setTouched(new Set(touched).add(f)),
  });
  const help = (f: SkillField, hint: string) => (
    <p id={`skill-${f}-help`} className={'sub' + (shown(f) ? ' problem' : '')} role={shown(f) ? 'alert' : undefined}>
      {shown(f) || hint}
    </p>
  );

  const submit = async () => {
    setAttempted(true);
    const first = SKILL_FIELDS.find((f) => problems[f]);
    if (first) return fields.current[first]?.focus();
    setSaving(true);
    try {
      const skill = await save({ ...draft, name: skillName(draft.name, true), description: draft.description.trim() });
      toast(`Saved ${skill.name}`);
      close();
    } finally {
      setSaving(false);
    }
  };
  const del = () =>
    confirm({
      title: `Delete ${draft.name}?`,
      text: 'The assistant can no longer use it. Chats where it was used stay as they are.',
      okLabel: 'Delete',
      ok: async () => {
        await remove(draft.id!);
        toast(`Deleted ${draft.name}`);
        close();
      },
    });

  const actions = (
    <>
      <button className="btn primary" disabled={saving} onClick={submit}>
        {draft.id ? 'Save changes' : 'Save skill'}
      </button>
      <button className="btn" onClick={close}>
        Cancel
      </button>
      {draft.id && (
        <button className="btn danger push-right" onClick={del}>
          Delete skill
        </button>
      )}
    </>
  );

  return (
    <SettingsLayout title={draft.id ? 'Edit skill' : 'New skill'} actions={actions}>
      <form
        className="skill-form"
        onSubmit={(e) => {
          e.preventDefault();
          void submit();
        }}
      >
        {draft.source && <p className="skill-source">{`Imported from ${draft.source}. Check it over, then save.`}</p>}
        <div className="field">
          <label htmlFor="skill-name">Name</label>
          <input className="input mono" autoComplete="off" spellCheck={false} autoFocus={!draft.id && !draft.source} placeholder="weekly-report" {...control('name', skillName)} />
          {help('name', 'Lowercase letters, numbers and hyphens. Spaces turn into hyphens as you type.')}
        </div>
        <div className="field">
          <label htmlFor="skill-description">
            Description
            {count > SKILL_DESCRIPTION_MAX_CHARS * SKILL_COUNT_FROM_SHARE && <span className="count">{`${count.toLocaleString()} of ${SKILL_DESCRIPTION_MAX_CHARS.toLocaleString()}`}</span>}
          </label>
          <textarea className="textarea short" dir="auto" rows={3} placeholder="Writes the weekly status report. Use when asked for a status update or a summary of the week." {...control('description')} />
          {help('description', 'What the skill does and when to use it. This is all the assistant sees until it loads the skill.')}
        </div>
        <div className="field">
          <label htmlFor="skill-instructions">Instructions</label>
          <textarea className="textarea mono tall" dir="auto" spellCheck={false} placeholder={'# Weekly report\n\n1. Start with what shipped.\n2. Then what is at risk, and why.\n3. Keep it to one page.'} {...control('instructions')} />
          {help('instructions', 'Markdown. The assistant reads this in full when a task calls for the skill.')}
        </div>
        <button type="submit" hidden />
      </form>
    </SettingsLayout>
  );
}
