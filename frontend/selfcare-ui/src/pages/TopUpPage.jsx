import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../services/api'

export default function TopUpPage() {
  const [amount, setAmount] = useState('')
  const [paymentMethod, setPaymentMethod] = useState('card')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const handleSubmit = async (e) => {
    e.preventDefault()
    setIsLoading(true)
    setError('')
    try {
      const { data } = await api.post('/topup', {
        amount: parseFloat(amount),
        currency: 'RUB',
        payment_method: paymentMethod,
        return_url: window.location.origin + '/topup/success',
      })
      if (data.payment_url) {
        window.location.href = data.payment_url
      }
    } catch (err) {
      setError(err.response?.data?.error?.message || 'Top-up failed')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="max-w-md mx-auto bg-white rounded-lg shadow p-6">
      <h1 className="text-2xl font-bold mb-6">Top Up Balance</h1>
      
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700">Amount</label>
          <input
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className="mt-1 w-full px-3 py-2 border rounded-md"
            placeholder="500"
            min="1"
            required
          />
        </div>
        
        <div>
          <label className="block text-sm font-medium text-gray-700">Payment Method</label>
          <select
            value={paymentMethod}
            onChange={(e) => setPaymentMethod(e.target.value)}
            className="mt-1 w-full px-3 py-2 border rounded-md"
          >
            <option value="card">Credit Card</option>
            <option value="crypto">Cryptocurrency</option>
          </select>
        </div>
        
        {error && <div className="text-red-600 text-sm">{error}</div>}
        
        <button
          type="submit"
          disabled={isLoading}
          className="w-full bg-brand-accent text-white py-2 rounded-md hover:opacity-90 disabled:opacity-50"
        >
          {isLoading ? 'Processing...' : 'Proceed to Payment'}
        </button>
      </form>
    </div>
  )
}
