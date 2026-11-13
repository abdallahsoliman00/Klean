/** Untrusted markup only ever runs inside a sandboxed frame with its own opaque origin:
    scripts may run, but there is no access to this page, its storage or the API. */
export function SandboxFrame({ doc, title }: { doc: string; title: string }) {
  return <iframe sandbox="allow-scripts" referrerPolicy="no-referrer" title={title} srcDoc={doc} />;
}
