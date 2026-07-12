import { Markdown } from './markdown';

// Renders a trusted Markdown doc (legal pages) via the shared pipeline.
export function MarkdownDoc({ markdown }: { markdown: string }) {
  return (
    <div className="markdown text-fg text-sm">
      <Markdown>{markdown}</Markdown>
    </div>
  );
}
