// Grow a textarea to its content; returns true once it wraps past line one.
export function autoSize(el: HTMLTextAreaElement): boolean {
  const cs = getComputedStyle(el);
  // add border back under border-box so zoom rounding can't add a scrollbar
  const borderY =
    cs.boxSizing === 'border-box'
      ? (parseFloat(cs.borderTopWidth) || 0) +
        (parseFloat(cs.borderBottomWidth) || 0)
      : 0;
  el.style.height = 'auto';
  const height = el.scrollHeight;
  el.style.height = `${height + borderY}px`;
  const oneLine =
    (parseFloat(cs.lineHeight) || parseFloat(cs.fontSize) * 1.5) +
    parseFloat(cs.paddingTop) +
    parseFloat(cs.paddingBottom);
  return height > oneLine + 1;
}
