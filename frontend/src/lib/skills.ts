/* The rules a skill must meet (the backend checks the same ones), as messages for the editor. */
import { SKILL_DESCRIPTION_MAX_CHARS, SKILL_FILE_MAX_BYTES, SKILL_INSTRUCTIONS_MAX_CHARS, SKILL_NAME_MAX_CHARS } from '../config';
import type { Skill, SkillInput } from '../types';
import { parseSkillFile, slugify, type SkillFile } from './skillFile';

/** What the user typed as a name, made valid as they type. */
export const skillName = (typed: string, done = false) => slugify(typed.slice(0, SKILL_NAME_MAX_CHARS * 2), done).slice(0, SKILL_NAME_MAX_CHARS);

export type SkillField = 'name' | 'description' | 'instructions';
export const SKILL_FIELDS: readonly SkillField[] = ['name', 'description', 'instructions'];

const tooLong = (what: string, max: number, has: number) => `Keep the ${what} to ${max.toLocaleString()} characters. It has ${has.toLocaleString()}.`;

/** What is wrong with each field, if anything. `others` are the skills this one must not share a name with. */
export function skillProblems(skill: SkillInput, others: Skill[]): Partial<Record<SkillField, string>> {
  const problems: Partial<Record<SkillField, string>> = {};
  const count = skill.description.trim().length;
  if (!skill.name) problems.name = 'Give the skill a name.';
  else if (others.some((s) => s.name.toLowerCase() === skill.name)) problems.name = `You already have a skill called ${skill.name}. Pick another name.`;
  if (!count) problems.description = 'Say when the assistant should use this skill.';
  else if (count > SKILL_DESCRIPTION_MAX_CHARS) problems.description = tooLong('description', SKILL_DESCRIPTION_MAX_CHARS, count);
  if (!skill.instructions.trim()) problems.instructions = 'Write the instructions the assistant should follow.';
  else if (skill.instructions.length > SKILL_INSTRUCTIONS_MAX_CHARS)
    problems.instructions = tooLong('instructions', SKILL_INSTRUCTIONS_MAX_CHARS, skill.instructions.length);
  return problems;
}

/** Reads a dropped or picked SKILL.md. Throws a message for the user when it isn't one. */
export async function readSkillFile(file: File): Promise<SkillFile> {
  if (!/\.(md|markdown)$/i.test(file.name) && !file.type.startsWith('text/')) throw new Error(`${file.name} isn't a Markdown file. Import a SKILL.md or another .md file.`);
  if (file.size > SKILL_FILE_MAX_BYTES) throw new Error(`${file.name} is too large to be a skill.`);
  const skill = parseSkillFile(await file.text(), file.name);
  if (!skill.instructions && !skill.description) throw new Error(`${file.name} is empty.`);
  return { ...skill, name: skillName(skill.name, true) };
}
