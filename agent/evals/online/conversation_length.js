// count user turns (conversation length) from a root run's inputs.
function performEval(run) {
  const msgs = (run.inputs && run.inputs.messages) || [];
  let turns = 0;
  for (const m of msgs) {
    if (!m) continue;
    if (m.type === "human" || m.role === "user") turns += 1;
  }
  return { conversation_length: turns };
}
