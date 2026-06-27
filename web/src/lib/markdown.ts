import type { Options } from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize';

// Highlight first (adds <span class="hljs-*">), then sanitize last as the
// security gate — extended to whitelist the highlight.js classes on code/span
// so the highlight markup survives. Restricting to hljs*/language-* (not a
// blanket className allow) keeps sanitization tight; a className can't execute.
const schema = {
  ...defaultSchema,
  attributes: {
    ...defaultSchema.attributes,
    code: [['className', /^language-./, /^hljs$/, /^hljs-/]],
    span: [['className', /^hljs$/, /^hljs-/]],
  },
};

export const remarkPlugins: Options['remarkPlugins'] = [remarkGfm];
export const rehypePlugins: Options['rehypePlugins'] = [
  [rehypeHighlight, { detect: true, ignoreMissing: true }],
  [rehypeSanitize, schema],
];
