import { describe, it, expect, afterEach, vi } from 'vitest';
import {
  render,
  screen,
  waitForElementToBeRemoved,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Menu } from './menu';

function setDesktop(isDesktop: boolean) {
  window.matchMedia = vi.fn(
    (query: string) =>
      ({
        matches: isDesktop,
        media: query,
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      }) as unknown as MediaQueryList,
  );
}

function Harness() {
  return (
    <Menu
      label="Actions"
      trigger={(p) => (
        <button {...p} aria-label="Open">
          menu
        </button>
      )}
    >
      {({ close }) => <button onClick={close}>Item</button>}
    </Menu>
  );
}

afterEach(() => vi.restoreAllMocks());

describe('Menu', () => {
  it('opens on trigger click and reflects aria-expanded', async () => {
    setDesktop(true);
    render(<Harness />);
    const trigger = screen.getByRole('button', { name: 'Open' });
    expect(trigger).toHaveAttribute('aria-expanded', 'false');
    await userEvent.click(trigger);
    expect(screen.getByRole('menu')).toBeInTheDocument();
    expect(trigger).toHaveAttribute('aria-expanded', 'true');
  });

  it('closes on Escape and returns focus to the trigger', async () => {
    setDesktop(true);
    render(<Harness />);
    const trigger = screen.getByRole('button', { name: 'Open' });
    await userEvent.click(trigger);
    await userEvent.keyboard('{Escape}');
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('closes on an outside click', async () => {
    setDesktop(true);
    render(<Harness />);
    await userEvent.click(screen.getByRole('button', { name: 'Open' }));
    await userEvent.click(document.body);
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('opens upward as a dialog when placement is top-start', async () => {
    setDesktop(true);
    render(
      <Menu
        label="Settings"
        placement="top-start"
        role="dialog"
        trigger={(p) => (
          <button {...p} aria-label="Open">
            menu
          </button>
        )}
      >
        {() => <button>Item</button>}
      </Menu>,
    );
    const trigger = screen.getByRole('button', { name: 'Open' });
    expect(trigger).toHaveAttribute('aria-haspopup', 'dialog');
    await userEvent.click(trigger);
    const surface = screen.getByRole('dialog');
    // anchored from the bottom (grows upward), not the top
    expect(surface.style.bottom).not.toBe('');
    expect(surface.style.top).toBe('');
  });

  it('renders a sheet with a backdrop on mobile and closes on backdrop tap', async () => {
    setDesktop(false);
    render(<Harness />);
    await userEvent.click(screen.getByRole('button', { name: 'Open' }));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('menu-backdrop'));
    await waitForElementToBeRemoved(() => screen.queryByRole('dialog'));
  });
});
