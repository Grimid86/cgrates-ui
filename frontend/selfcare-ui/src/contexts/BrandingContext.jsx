import { createContext, useContext, useEffect, useState } from 'react'
import api from '../services/api'

const BrandingContext = createContext(null)

export function BrandingProvider({ children }) {
  const [branding, setBranding] = useState(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    api.get('/branding?domain=' + window.location.host)
      .then(({ data }) => {
        setBranding(data)
        if (data.css_variables) {
          Object.entries(data.css_variables).forEach(([key, value]) => {
            document.documentElement.style.setProperty(key, value)
          })
        }
        if (data.product_name) {
          document.title = data.product_name
        }
      })
      .catch(() => setBranding({ product_name: 'SelfCare' }))
      .finally(() => setIsLoading(false))
  }, [])

  return (
    <BrandingContext.Provider value={{ branding, isLoading }}>
      {children}
    </BrandingContext.Provider>
  )
}

export const useBranding = () => useContext(BrandingContext)
