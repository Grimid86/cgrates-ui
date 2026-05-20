import { useQuery } from 'react-query'
import { useAuth } from '../contexts/AuthContext'
import { useI18n } from '../contexts/I18nContext'
import api from '../services/api'
import Card from '../components/ui/Card'
import BalanceChart from '../components/charts/BalanceChart'
import CDRChart from '../components/charts/CDRChart'

export default function DashboardPage() {
  const { user } = useAuth()
  const { t } = useI18n()

  const { data: balanceData, isLoading: balanceLoading } = useQuery('balance', () => api.get('/balance'))
  const { data: cdrData, isLoading: cdrLoading } = useQuery('cdr-dashboard', () => api.get('/cdr?per_page=50'))

  const balances = balanceData?.data?.balances || []
  const monetary = balances.find((b) => b.type === '*monetary')
  const dataBalance = balances.find((b) => b.type === '*data')
  const voice = balances.find((b) => b.type === '*voice')

  const cdrs = cdrData?.data?.data || []

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">
          {t('app.title', 'common')}
        </h1>
        <p className="text-gray-500">{user?.msisdn}</p>
      </div>

      {/* Balance Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card
          title={t('balance.monetary', 'balance')}
          value={monetary ? `${monetary.value.toFixed(2)} ${monetary.currency || '₽'}` : '0.00'}
          subtitle={monetary?.expiry_date ? `Expires: ${monetary.expiry_date}` : 'No expiry'}
          icon="💰"
          color="primary"
        />
        <Card
          title={t('balance.data', 'balance')}
          value={dataBalance ? `${(dataBalance.value / 1024).toFixed(1)} GB` : '0 GB'}
          subtitle={dataBalance?.expiry_date ? `Expires: ${dataBalance.expiry_date}` : 'No expiry'}
          icon="📶"
          color="success"
        />
        <Card
          title={t('balance.voice', 'balance')}
          value={voice ? `${voice.value} ${voice.unit || 'min'}` : '0 min'}
          subtitle={voice?.expiry_date ? `Expires: ${voice.expiry_date}` : 'No expiry'}
          icon="📞"
          color="warning"
        />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Balance Distribution</h2>
          <BalanceChart balances={balances} />
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Recent Activity</h2>
          <CDRChart cdrs={cdrs} />
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Quick Actions</h2>
        <div className="flex gap-4">
          <a
            href="/topup"
            className="inline-flex items-center px-4 py-2 bg-brand-accent text-white rounded-lg hover:opacity-90"
          >
            Top Up Balance
          </a>
          <a
            href="/cdr"
            className="inline-flex items-center px-4 py-2 bg-brand-primary text-white rounded-lg hover:opacity-90"
          >
            View History
          </a>
        </div>
      </div>
    </div>
  )
}
