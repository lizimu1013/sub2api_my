interface APIErrorLike {
  message?: string
  response?: {
    data?: {
      detail?: string
      message?: string
      reason?: string
    }
  }
}

const AUTH_ERROR_MESSAGES_BY_REASON: Record<string, string> = {
  REGISTRATION_IP_BLACKLISTED: '当前暂不允许注册',
}

function extractErrorMessage(error: unknown): string {
  const err = (error || {}) as APIErrorLike
  const reason = err.response?.data?.reason || ''
  if (reason && AUTH_ERROR_MESSAGES_BY_REASON[reason]) {
    return AUTH_ERROR_MESSAGES_BY_REASON[reason]
  }
  return err.response?.data?.detail || err.response?.data?.message || err.message || ''
}

export function buildAuthErrorMessage(
  error: unknown,
  options: {
    fallback: string
  }
): string {
  const { fallback } = options
  const message = extractErrorMessage(error)
  return message || fallback
}
