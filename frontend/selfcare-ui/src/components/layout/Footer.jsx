import { useBranding } from '../../contexts/BrandingContext'

export default function Footer() {
  const { branding } = useBranding()

  return (
    <footer className="bg-gray-800 text-gray-300 py-4">
      <div className="container mx-auto px-4 text-center text-sm">
        <p>&copy; {new Date().getFullYear()} {branding?.product_name || 'CGRateS Billing'}</p>
        {branding?.support_email && (
          <p className="mt-1">Support: {branding.support_email}</p>
        )}
      </div>
    </footer>
  )
}
