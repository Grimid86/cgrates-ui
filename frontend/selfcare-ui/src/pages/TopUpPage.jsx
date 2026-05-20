import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../services/api'
import Card from '../components/ui/Card'

const AMOUNTS = [100, 300, 500, 1000, 2000, 5000]

export default function TopUpPage() {
  const [amount, setAmount] = useState('')
  const [customAmount, setCustomAmount] = useState('')
  const [paymentMethod, setPaymentMethod] = useState('card')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const handleSelectAmount = (val) => {
    setAmount(String(val))
    setCustomAmount('')
  }

  const handleCustomAmount = (val) => {
    setCustomAmount(val)
    setAmount(val)
  }

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
    <div className="max-w-lg mx-auto space-y-6">
      <h1 className="text-2xl font-bold">Top Up Balance</h1>

      <div className="grid grid-cols-3 gap-3">
        {AMOUNTS.map((val) => (
          <button
            key={val}
            onClick={() => handleSelectAmount(val)}
            className={`p-4 rounded-lg border-2 text-center transition ${
              amount === String(val)
                ? 'border-brand-primary bg-blue-50 text-brand-primary'
                : 'border-gray-200 hover:border-gray-300'
            }`}
          >
            <div className="text-xl font-bold">{val}</div>
            <div className="text-xs text-gray-500">RUB</div>
          </button>
        ))}
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Or enter custom amount</label>
            <div className="mt-1 relative">
              <input
                type="number"
                value={customAmount}
                onChange={(e) => handleCustomAmount(e.target.value)}
                className="w-full px-3 py-2 border rounded-md pr-12"
                placeholder="0"
                min="1"
              />
              <span className="absolute right-3 top-2 text-gray-400">₽</span>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700">Payment Method</label>
            <div className="mt-2 space-y-2">
              <label className="flex items-center p-3 border rounded-lg cursor-pointer hover:bg-gray-50">
                <input
                  type="radio"
                  value="card"
                  checked={paymentMethod === 'card'}
                  onChange={(e) => setPaymentMethod(e.target.value)}
                  className="mr-3"
                />
                <span>💳 Credit Card</span>
              </label>
              <label className="flex items-center p-3 border rounded-lg cursor-pointer hover:bg-gray-50">
                <input
                  type="radio"
                  value="crypto"
                  checked={paymentMethod === 'crypto'}
                  onChange={(e) => setPaymentMethod(e.target.value)}
                  className="mr-3"
                />
                <span>₿ Cryptocurrency</span>
              </label>
            </div>
          </div>

          {error && <div className="text-red-600 text-sm">{error}</div>}

          <div className="pt-2">
            <div className="flex justify-between text-sm text-gray-600 mb-2">
              <span>Amount</span>
              <span>{amount ? `${amount} ₽` : '0 ₽'}</span>
            </div>
            <div className="flex justify-between text-sm text-gray-600 mb-4">
              <span>Fee</span>
              <span>0 ₽</span>
            </div>
            <div className="flex justify-between font-semibold text-lg mb-4">
              <span>Total</span>
              <span>{amount ? `${amount} ₽` : '0 ₽'}</span>
            </div>
          </div>

          <button
            type="submit"
            disabled={isLoading || !amount}
            className="w-full bg-brand-accent text-white py-3 rounded-lg hover:opacity-90 disabled:opacity-50 font-semibold"
          >
            {isLoading ? 'Processing...' : 'Proceed to Payment'}
          </button>
        </form>
      </div>
    </div>
  )
}
