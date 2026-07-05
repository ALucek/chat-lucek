// Grow a textarea to its content; returns true once it wraps past line one.
export function autoSize(el: HTMLTextAreaElement): boolean {
  el.style.height = 'auto';
  const height = el.scrollHeight;
  el.style.height = `${height}px`;
  const cs = getComputedStyle(el);
  const oneLine =
    (parseFloat(cs.lineHeight) || parseFloat(cs.fontSize) * 1.5) +
    parseFloat(cs.paddingTop) +
    parseFloat(cs.paddingBottom);
  return height > oneLine + 1;
}
