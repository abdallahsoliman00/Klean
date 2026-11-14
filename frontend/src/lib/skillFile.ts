/* SKILL.md files: YAML frontmatter with `name` and `description`, then the instructions as markdown.
   No imports, so tests/skillFile.test.mjs can run it under plain Node. */

/** "My Skill" → "my-skill": lowercase letters, digits and hyphens only. A trailing hyphen is kept
    while typing (the next word is coming); pass `done` to trim the ends. */
export function slugify(text: string, done = false): string {
  const slug = text
    .normalize('NFKD')
    .toLowerCase()
    .replace(/[\s_.]+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
    .replace(/-{2,}/g, '-')
    .replace(/^-/, '');
  return done ? slug.replace(/-$/, '') : slug;
}

function unquote(value: string): string {
  if (/^"[\s\S]*"$/.test(value)) return value.slice(1, -1).replace(/\\(["\\])/g, '$1');
  if (/^'[\s\S]*'$/.test(value)) return value.slice(1, -1).replace(/''/g, "'");
  return value.replace(/\s+#.*$/, ''); // a trailing comment
}

/** The top-level `key: value` pairs of a frontmatter block. A value may be quoted, continue on indented
    lines, or be a block scalar (`|` keeps line breaks, `>` joins lines). Nested keys are left alone. */
function frontmatter(source: string): Record<string, string> {
  const out: Record<string, string> = {};
  let key = '';
  let join = ' ';
  for (const line of source.split(/\r?\n/)) {
    const pair = /^([A-Za-z_][\w-]*):(?:[ \t]+(.*))?$/.exec(line);
    if (pair) {
      key = pair[1];
      const value = (pair[2] ?? '').trim();
      const block = /^[|>][+-]?$/.test(value);
      join = value.startsWith('|') ? '\n' : ' ';
      out[key] = block ? '' : unquote(value);
    } else if (key && /^[ \t]+\S/.test(line)) {
      out[key] += (out[key] ? join : '') + line.trim();
    }
  }
  return out;
}

export interface SkillFile {
  name: string;
  description: string;
  instructions: string;
}

/** Reads a SKILL.md. Without frontmatter the whole file is the instructions, and the name comes from the
    file's own name (unless that is just "SKILL.md"). */
export function parseSkillFile(text: string, filename = ''): SkillFile {
  const head = /^\uFEFF?---[ \t]*\r?\n([\s\S]*?)\r?\n---[ \t]*(?:\r?\n|$)/.exec(text);
  const meta = head ? frontmatter(head[1]) : {};
  const stem = filename.replace(/\.[^.]*$/, '');
  return {
    name: slugify(meta.name || (/^skill$/i.test(stem) ? '' : stem), true),
    description: (meta.description ?? '').trim(),
    instructions: (head ? text.slice(head[0].length) : text.replace(/^\uFEFF/, '')).trim(),
  };
}
