// Run with `npm test` (Node strips the types from the .ts it imports).
import assert from 'node:assert/strict';
import { test } from 'node:test';
import { parseSkillFile, slugify } from '../src/lib/skillFile.ts';

test('slugify keeps to lowercase letters, digits and hyphens', () => {
  assert.equal(slugify('My Skill'), 'my-skill');
  assert.equal(slugify('my '), 'my-'); // still typing
  assert.equal(slugify('my ', true), 'my');
  assert.equal(slugify('  PDF_Reports v2.1!'), 'pdf-reports-v2-1');
  assert.equal(slugify('Café --- notes'), 'cafe-notes');
});

test('reads name, description and body from frontmatter', () => {
  const file = '---\nname: Code Review\ndescription: "Review a diff: bugs first, then style."\nlicense: MIT\n---\n\n# Steps\n\n1. Read it.\n';
  assert.deepEqual(parseSkillFile(file, 'SKILL.md'), {
    name: 'code-review',
    description: 'Review a diff: bugs first, then style.',
    instructions: '# Steps\n\n1. Read it.',
  });
});

test('reads folded, literal and continued values, and CRLF files', () => {
  const file = '---\r\nname: weekly-report\r\ndescription: >\r\n  Write the weekly report.\r\n  Use when asked for a status update.\r\nmetadata:\r\n  author: someone\r\n---\r\nBody';
  const skill = parseSkillFile(file);
  assert.equal(skill.description, 'Write the weekly report. Use when asked for a status update.');
  assert.equal(skill.instructions, 'Body');
  assert.equal(parseSkillFile('---\nname: a\ndescription: |\n  one\n  two\n---\nx').description, 'one\ntwo');
  assert.equal(parseSkillFile("---\nname: a\ndescription: it's\n  continued\n---\nx").description, "it's continued");
});

test('a file without frontmatter is all instructions, named after the file', () => {
  assert.deepEqual(parseSkillFile('Just do it.\n', 'Tone Guide.md'), { name: 'tone-guide', description: '', instructions: 'Just do it.' });
  assert.equal(parseSkillFile('--- not frontmatter', 'SKILL.md').name, '');
});
