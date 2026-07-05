import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MessageList } from './message-list';
import type { ChatMessage } from '@/lib/messages-context';
import type { RunNode } from '@/lib/api';

describe('MessageList markdown', () => {
  it('renders assistant markdown (bold, link, list)', () => {
    const msgs: ChatMessage[] = [
      {
        id: 1,
        role: 'assistant',
        content: '**bold** and [x](https://e.com)\n\n- a\n- b',
        created_at: '',
      },
    ];
    render(<MessageList messages={msgs} />);
    expect(screen.getByText('bold').tagName).toBe('STRONG');
    const link = screen.getByRole('link', { name: 'x' });
    expect(link).toHaveAttribute('href', 'https://e.com');
    expect(link).toHaveAttribute('target', '_blank');
    expect(screen.getByText('a')).toBeInTheDocument();
    expect(screen.getByText('b')).toBeInTheDocument();
  });

  it('lets long content wrap instead of overflowing', () => {
    const msgs: ChatMessage[] = [
      { id: 1, role: 'user', content: 'hello', created_at: '' },
    ];
    render(<MessageList messages={msgs} />);
    const bubble = screen.getByText('hello');
    expect(bubble.className).toContain('break-words');
    expect(bubble.className).toContain('min-w-0');
  });

  it('renders a user message as plain text, not markdown', () => {
    const msgs: ChatMessage[] = [
      { id: 1, role: 'user', content: '**not bold**', created_at: '' },
    ];
    render(<MessageList messages={msgs} />);
    expect(screen.getByText('**not bold**')).toBeInTheDocument();
    expect(screen.queryByRole('strong')).toBeNull();
  });

  it('neutralizes a javascript: link and raw HTML', () => {
    const msgs: ChatMessage[] = [
      {
        id: 1,
        role: 'assistant',
        content:
          '[click](javascript:alert(1))\n\n<img src=x onerror="alert(1)">',
        created_at: '',
      },
    ];
    render(<MessageList messages={msgs} />);
    const link = screen.queryByRole('link', { name: 'click' });
    if (link)
      expect(link.getAttribute('href') ?? '').not.toMatch(/^javascript:/i);
    // raw HTML is not parsed
    expect(document.querySelector('img')).toBeNull();
  });

  it('highlights fenced code blocks and keeps sanitization', () => {
    const msgs: ChatMessage[] = [
      {
        id: 1,
        role: 'assistant',
        content: '```go\nfunc main() {}\n```',
        created_at: '',
      },
    ];
    render(<MessageList messages={msgs} />);
    const code = document.querySelector('pre code');
    expect(code?.className).toMatch(/hljs/);
    expect(document.querySelector('.hljs-keyword')).not.toBeNull();
  });
});

describe('MessageList timeline', () => {
  it('renders steps and answer from the node log, subagent folded', () => {
    const nodes: RunNode[] = [
      { id: 'r', parent_id: null, type: 'reasoning', text: 'hmm' },
      {
        id: 'SA',
        parent_id: null,
        type: 'tool',
        name: 'run_subagent',
        input: { task: 'dig' },
      },
      {
        id: 's1',
        parent_id: 'SA',
        type: 'tool',
        name: 'internet_search',
        input: { query: 'q' },
      },
      { id: 'a', parent_id: null, type: 'text', text: 'the answer' },
    ];
    render(
      <MessageList
        messages={[
          { id: 1, role: 'assistant', content: '', created_at: '', nodes },
        ]}
      />,
    );
    expect(screen.getByText('thinking')).toBeInTheDocument();
    expect(screen.getByText('subagent')).toBeInTheDocument();
    expect(screen.getByText('the answer')).toBeInTheDocument();
    // subagent is collapsed by default, so its nested search is hidden
    expect(screen.queryByText('search')).toBeNull();
  });

  it('marks an in-flight subagent as running while the turn streams', () => {
    const nodes: RunNode[] = [
      {
        id: 'SA',
        parent_id: null,
        type: 'tool',
        name: 'run_subagent',
        input: { task: 'go' },
      },
      {
        id: 's1',
        parent_id: 'SA',
        type: 'tool',
        name: 'internet_search',
        input: { query: 'q' },
      },
    ];
    const { container } = render(
      <MessageList
        messages={[
          {
            id: 1,
            role: 'assistant',
            content: '',
            created_at: '',
            streaming: true,
            nodes,
          },
        ]}
      />,
    );
    // subagent has no output yet + turn is streaming -> a running dot shows
    expect(container.querySelector('.animate-pulse')).not.toBeNull();
  });

  it('shows a streaming caret on the last answer node', () => {
    const nodes: RunNode[] = [
      { id: 'a', parent_id: null, type: 'text', text: 'partial' },
    ];
    const { container } = render(
      <MessageList
        messages={[
          {
            id: 1,
            role: 'assistant',
            content: '',
            created_at: '',
            streaming: true,
            nodes,
          },
        ]}
      />,
    );
    expect(container.querySelector('.caret-blink')).not.toBeNull();
  });

  it('drops the caret from a preamble once a tool follows it', () => {
    const nodes: RunNode[] = [
      { id: 'p', parent_id: null, type: 'text', text: 'let me look' },
      {
        id: 'SA',
        parent_id: null,
        type: 'tool',
        name: 'run_subagent',
        input: { task: 'dig' },
      },
      {
        id: 's1',
        parent_id: 'SA',
        type: 'tool',
        name: 'internet_search',
        input: { query: 'q' },
      },
    ];
    const { container } = render(
      <MessageList
        messages={[
          {
            id: 1,
            role: 'assistant',
            content: '',
            created_at: '',
            streaming: true,
            nodes,
          },
        ]}
      />,
    );
    // last node is the subagent, so the preamble text has no live caret
    expect(container.querySelector('.caret-blink')).toBeNull();
  });
});
