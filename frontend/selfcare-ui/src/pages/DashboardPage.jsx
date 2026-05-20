import { useAuth } from '../contexts/AuthContext'
import { useI18n } from '../contexts/I18nContext'

export default function DashboardPage() {
  const { user } = useAuth()
  const { t } = useI18n()

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">
        Welcome, {user?.msisdn || 'Subscriber'}
      </h1>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="text-gray-500 text-sm">{t('balance.monetary', 'balance')}</h3>
          <p className="text-2xl font-bold text-brand-primary">150.50 ₽</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="text-gray-500 text-sm">{t('balance.data', 'balance')}</h3>
          <p className="text-2xl font-bold text-brand-accent">5.1 GB</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="text-gray-500 text-sm">{t('balance.voice', 'balance')}</h3>
          <p className="text-2xl font-bold text-brand-secondary">300 min</p>
        </div>
      </div>
    </div>
  )
}
