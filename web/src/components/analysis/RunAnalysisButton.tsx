import { Loader2, Sparkles } from 'lucide-react'
import { useAnalysis } from '../../lib/analysis'
import { Button } from '../ui/Button'

interface RunAnalysisButtonProps {
  meterIds?: string[]
  label?: string
  size?: 'sm' | 'md'
  className?: string
}

export function RunAnalysisButton({ meterIds, label = 'Run AI Analysis', size = 'md', className }: RunAnalysisButtonProps) {
  const { start, running, openPanel } = useAnalysis()
  return (
    <Button
      variant="primary"
      size={size}
      className={className}
      aria-live="polite"
      icon={running ? <Loader2 className="size-4 animate-spin" aria-hidden /> : <Sparkles className="size-4" aria-hidden />}
      onClick={() => (running ? openPanel() : start(meterIds))}
    >
      {running ? 'Analizando… ver progreso' : label}
    </Button>
  )
}
