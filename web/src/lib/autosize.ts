// Grow a textarea to fit its content; CSS max-height caps it, then it scrolls.
export function autoSize(el: HTMLTextAreaElement) {
  el.style.height = 'auto';
  el.style.height = `${el.scrollHeight}px`;
}
