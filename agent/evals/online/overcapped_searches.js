// count over-cap search calls in a subagent run.
function performEval(run) {
  const msgs = (run.outputs && run.outputs.messages) || [];
  let count = 0;
  for (const m of msgs) {
    if (!m || m.type !== "tool") continue;
    const c =
      typeof m.content === "string" ? m.content : JSON.stringify(m.content || "");
    if (c.includes("Search limit reached")) count += 1;
  }
  return { overcapped_searches: count };
}
