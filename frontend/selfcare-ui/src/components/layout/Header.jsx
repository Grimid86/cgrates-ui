import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { useBranding } from '../../contexts/BrandingContext'
import { useI18n } from '../../contexts/I18nContext'

export default function Header() {
  const { user, logout, isAuthenticated } = useAuth()
  const { branding } = useBranding()
  const { locale, setLocale, t } = useI18n()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <header className="bg-brand-primary text-white shadow-md">
      <div className="container mx-auto px-4 py-3 flex items-center justify-between">
        <Link to="/" className="text-xl font-bold">
          {branding?.product_name || 'SelfCare'}
        </Link>
        
        <nav className="flex items-center gap-4">
          {isAuthenticated && (
            <>
              <Link to="/balance" className="hover:opacity-80">{t('balance.monetary', 'balance')}</Link>
              <Link to="/cdr" className="hover:opacity-80">{t('app.title', 'common')}</Link>
              <Link to="/profile" className="hover:opacity-80">Profile</Link>
            </>
          )}
          
          <select 
            value={locale} 
            onChange={(e) => setLocale(e.target.value)}
            className="bg-white/20 rounded px-2 py-1 text-sm"
          >
            <option value="en">EN</option>
            <option value="ru">RU</option>
            <option value="es">ES</option>
          </select>
          
          {isAuthenticated ? (
            <button onClick={handleLogout} className="hover:opacity-80">
              Logout
            </button>
          ) : (
            <Link to="/login" className="hover:opacity-80">Login</Link>
          )}
        </nav>
      </div>
    </header>
  )
}
