import { createContext, useContext, useEffect, useState } from 'react'
import i18n from '../i18n/config'
import api from '../services/api'

const I18nContext = createContext(null)

export function I18nProvider({ children }) {
  const [locale, setLocale] = useState(localStorage.getItem('selfcare_locale') || 'en')
  const [translations, setTranslations] = useState({})

  useEffect(() => {
    api.get(`/translations/${locale}`)
      .then(({ data }) => {
        setTranslations(data.translations || {})
        localStorage.setItem('selfcare_locale', locale)
      })
  }, [locale])

  const t = (key, category = 'common') => {
    return translations[category]?.[key] || key
  }

  return (
    <I18nContext.Provider value={{ locale, setLocale, t, translations }}>
      {children}
    </I18nContext.Provider>
  )
}

export const useI18n = () => useContext(I18nContext)
