import { Compass } from 'lucide-react'
import { Link } from 'react-router-dom'
import { EmptyState } from '../components/ui/Feedback'
import { GlassCard } from '../components/ui/GlassCard'

export function NotFoundPage() {
  return (
    <GlassCard dense>
      <EmptyState
        icon={<Compass className="size-6" aria-hidden />}
        title="Página no encontrada"
        description="La ruta que buscas no existe o fue movida."
        action={
          <Link to="/" className="text-sm font-semibold text-aurora-cyan hover:underline">
            Volver al dashboard
          </Link>
        }
      />
    </GlassCard>
  )
}
