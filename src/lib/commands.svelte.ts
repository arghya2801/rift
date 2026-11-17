export type Command = { id: string; title: string; run: () => unknown };

const groups = $state<Record<string, Command[]>>({});

export function registerCommands(id: string, cmds: Command[]) {
  groups[id] = cmds;
}

export function unregister(id: string) {
  delete groups[id];
}

export function commands(): Command[] {
  return Object.values(groups).flat();
}
