import assert from 'node:assert/strict';
import { test } from 'node:test';
import { parseBlocks, serializeBlocks, splitAtFence } from '../src/lib/composerBlocks.ts';

const t = (value) => ({ code: false, lang: '', value });
const c = (lang, value) => ({ code: true, lang, value });

test('closed fences become code blocks between text blocks', () => {
  assert.deepEqual(parseBlocks('see\n```py\nx = 1\n\ny\n```\nok?'), [t('see'), c('py', 'x = 1\n\ny'), t('ok?')]);
  assert.deepEqual(parseBlocks('```\na\n```\n```js\nb\n```'), [t(''), c('', 'a'), t(''), c('js', 'b'), t('')]);
  assert.deepEqual(parseBlocks('```py\nstill typing'), [t('```py\nstill typing')]);
  assert.deepEqual(parseBlocks(''), [t('')]);
});

test('blocks make the message they came from', () => {
  for (const m of ['see\n```py\nx = 1\n\ny\n```\nok?', '```\na\n```\n```js\nb\n```', 'plain\n\ntext']) {
    assert.equal(serializeBlocks(parseBlocks(m)), m);
  }
});

test('Enter at the end of a fence line splits around it', () => {
  assert.deepEqual(splitAtFence('hi\n```go', 8), { before: 'hi', lang: 'go', after: '' });
  assert.deepEqual(splitAtFence('```\nafter', 3), { before: '', lang: '', after: 'after' });
  assert.deepEqual(splitAtFence('a\n```\nb', 5), { before: 'a', lang: '', after: 'b' });
  assert.equal(splitAtFence('hi ```go', 8), null);
  assert.equal(splitAtFence('```go', 3), null); // caret mid-line
});
