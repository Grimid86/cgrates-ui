import { createContext, useContext, useEffect, useState } from 'react'
import api from '../services/api'
const BrandingContext = createContext(null)
export function BrandingProvider({ children }) {
  const [branding, setBranding] = useState(null)
  useEffect(() => {
    api.get('/branding?domain=' + window.location.host).then(({ data }) => setBranding(data)).catch(() => setBranding({}))
  }, [])
  return <BrandingContext.Provider value={{ branding }}>{children}</BrandingContext.Provider>
}
export const useBranding = () => useContext(BrandingContext)
