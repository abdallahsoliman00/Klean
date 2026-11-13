/** Shown instead of the thread when the backend didn't answer at startup. */
export function BackendDown({ message }: { message: string }) {
  return (
    <div className="welcome">
      <h1>The backend isn’t answering</h1>
      <p>{message} Start it, then reload this page.</p>
      <button className="btn primary" onClick={() => location.reload()}>
        Reload
      </button>
    </div>
  );
}
