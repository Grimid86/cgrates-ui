import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from 'react-query'
import api from '../services/api'

export default function LocalizationPage() {
  const [locale, setLocale] = useState('')
  const [translations, setTranslations] = useState('')
  const queryClient = useQueryClient()

  const { data: langsData } = useQuery('languages', () => api.get('/languages'))
  const languages = langsData?.data || []

  const { data: transData } = useQuery(
    ['translations', locale],
    () => api.get(`/translations/${locale}`),
    { enabled: !!locale }
  )

  const uploadMutation = useMutation(
    (data) => api.post('/translations', data),
    { onSuccess: () => queryClient.invalidateQueries(['translations', locale]) }
  )

  const handleUpload = () => {
    try {
      const parsed = JSON.parse(translations)
      uploadMutation.mutate({ locale, translations: parsed })
    } catch (e) {
      alert('Invalid JSON')
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Localization</h1>

      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Languages</h2>
        <div className="flex gap-2 flex-wrap">
          {languages.map((l) => (
            <button key={l.code} onClick={() => setLocale(l.code)} className={`px-3 py-1 rounded text-sm ${locale === l.code ? 'bg-blue-600 text-white' : 'bg-gray-100'}`}>
              {l.name} ({l.code})
            </button>
          ))}
        </div>
      </div>

      {locale && (
        <div className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Translations for {locale}</h2>
          <textarea
            value={translations || JSON.stringify(transData?.data || {}, null, 2)}
            onChange={(e) => setTranslations(e.target.value)}
            className="w-full h-64 px-4 py-2 border rounded-lg text-sm font-mono"
          />
          <button onClick={handleUpload} disabled={uploadMutation.isLoading} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50">
            {uploadMutation.isLoading ? 'Uploading...' : 'Upload'}
          </button>
        </div>
      )}
    </div>
  )
}
