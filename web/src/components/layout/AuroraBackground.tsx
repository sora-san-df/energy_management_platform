export function AuroraBackground() {
  return (
    <div className="pointer-events-none fixed inset-0 -z-10 overflow-hidden bg-[#0F172A]" aria-hidden>
      <div className="aurora-orb absolute -left-40 -top-40 size-[38rem] rounded-full bg-[#06B6D4]/25 blur-[120px]" />
      <div className="aurora-orb absolute -right-32 top-1/4 size-[34rem] rounded-full bg-[#8B5CF6]/25 blur-[130px] [animation-delay:-7s]" />
      <div className="aurora-orb absolute -bottom-48 left-1/3 size-[30rem] rounded-full bg-[#06B6D4]/15 blur-[120px] [animation-delay:-14s]" />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,transparent_0%,#0F172A_85%)]" />
    </div>
  )
}
