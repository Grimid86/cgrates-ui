import { useQuery } from 'react-query'
import api from '../services/api'

export default function RBACPage() {
  const { data: rolesData } = useQuery('roles', () => api.get('/rbac/roles'))
  const { data: permsData } = useQuery('permissions', () => api.get('/rbac/permissions'))
  const { data: matrixData } = useQuery('matrix', () => api.get('/rbac/matrix'))

  const roles = rolesData?.data?.roles || []
  const permissions = permsData?.data?.permissions || []
  const matrix = matrixData?.data?.matrix || {}

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">RBAC Matrix</h1>

      <div className="bg-white rounded-lg shadow overflow-auto">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left sticky left-0 bg-gray-50">Permission</th>
              {roles.map((r) => <th key={r.code} className="px-4 py-3 text-center text-sm">{r.code}</th>)}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {permissions.map((p) => (
              <tr key={p.code} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm sticky left-0 bg-white font-medium">{p.code}</td>
                {roles.map((r) => (
                  <td key={r.code} className="px-4 py-3 text-center">
                    {matrix[r.code]?.[p.code] ? (
                      <span className="text-green-600">✓</span>
                    ) : (
                      <span className="text-gray-300">—</span>
                    )}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
