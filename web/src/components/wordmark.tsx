const ART = [
  ' __                  _       ____ _           _   ',
  '| //  _   _  ___ ___| | __  / ___| |__   __ _| |_ ',
  "|//| | | | |/ __/ _ \\ |/ / | |   | '_ \\ / _` | __|",
  '// |_| |_| | (_|  __/   <  | |___| | | | (_| | |_ ',
  '|_____\\__,_|\\___\\___|_|\\_\\  \\____|_| |_|\\__,_|\\__|',
].join('\n');

export function Wordmark() {
  return (
    <pre
      aria-label="Łucek Chat"
      style={{ fontFamily: '"Courier New", monospace' }}
      className="text-fg-strong overflow-x-auto text-center text-[clamp(7px,2.5vw,17px)] leading-[1.2]"
    >
      {ART}
    </pre>
  );
}
