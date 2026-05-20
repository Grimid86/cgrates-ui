import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from 'react-query'
import api from '../services/api'

export default function BrandingPage() {
  const [cssVars, setCssVars] = useState('')
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery('branding', () => api.get('/branding'))
  const branding = data?.data || {}

  const updateMutation = useMutation(
    (data) => api.put('/branding', data),
    { onSuccess: () => queryClient.invalidateQueries('branding') }
  )

  const handleUpdate = () => {
    try {
      const parsed = JSON.parse(cssVars)
      updateMutation.mutate(parsed)
    } catch (e) {
      alert('Invalid JSON')
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">White-Label Branding</h1>

      {isLoading ? <div>Loading...</div> : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold mb-4">Current Config</h2>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between"><span>Product Name:</span> <span className="font-medium">{branding.product_name}</span></div>
              <div className="flex justify-between"><span>Primary Color:</span> <span className="font-medium flex items-center gap-2"><span className="w-4 h-4 rounded inline-block" style={{background: branding.primary_color}}></span>{branding.primary_color}</span></div>
              <div className="flex justify-between"><span>Secondary Color:</span> <span className="font-medium flex items-center gap-2"><span className="w-4 h-4 rounded inline-block" style={{background: branding.secondary_color}}></span>{branding.secondary_color}</span></div>
              <div className="flex justify-between"><span>Logo URL:</span> <span className="font-medium">{branding.logo_url || 'N/A'}</span></div>
              <div className="flex justify-between"><span>Support Email:</span> <span className="font-medium">{branding.support_email || 'N/A'}</span></div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold mb-4">Update CSS Variables</h2>
            <textarea
              value={cssVars}
              onChange={(e) => setCssVars(e.target.value)}
              placeholder='{"primary_color": "#2563eb", "secondary_color": "#1e40af"}'
              className="w-full h-32 px-4 py-2 border rounded-lg text-sm font-mono"
            />
            <button onClick={handleUpdate} disabled={updateMutation.isLoading} className="mt-4 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50">
              {updateMutation.isLoading ? 'Saving...' : 'Update'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
