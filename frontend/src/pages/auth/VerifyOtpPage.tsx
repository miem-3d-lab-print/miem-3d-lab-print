import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { KeyRound } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { Navigate, useNavigate } from 'react-router-dom';
import { z } from 'zod';
import { authApi } from '../../api/endpoints';
import { useAuth } from '../../auth/AuthContext';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { normalizeApiError } from '../../utils/errors';

const schema = z.object({ code: z.string().regex(/^\d{6}$/, 'Введите 6 цифр') });
type FormValues = z.infer<typeof schema>;

export function VerifyOtpPage() {
  const email = sessionStorage.getItem('miem3d_pending_email');
  const navigate = useNavigate();
  const { establishSession } = useAuth();
  const { register, handleSubmit, formState: { errors } } = useForm<FormValues>({ resolver: zodResolver(schema) });
  const verify = useMutation({
    mutationFn: (code: string) => authApi.verifyOtp(email!, code),
    onSuccess: async (response) => {
      const profile = await establishSession(response);
      sessionStorage.removeItem('miem3d_pending_email');
      sessionStorage.removeItem('miem3d_otp_expires_in');
      const afterAuth = sessionStorage.getItem('miem3d_after_auth');
      if (!profile.consent_given || response.consent_required) {
        navigate('/consent', { replace: true, state: { afterAuth } });
      } else if (!profile.full_name || (!profile.telegram && !profile.max)) {
        navigate('/profile', { replace: true, state: { profileRequired: true, afterAuth } });
      } else {
        sessionStorage.removeItem('miem3d_after_auth');
        navigate(afterAuth || '/app/applications', { replace: true });
      }
    },
  });
  if (!email) return <Navigate to="/auth/login" replace />;
  const apiError = verify.isError ? normalizeApiError(verify.error) : null;
  const lockedUntil = apiError?.details?.locked_until;
  const attemptsLeft = apiError?.details?.attempts_left;

  return <section className="auth-page"><Card className="auth-card"><div className="auth-icon"><KeyRound /></div><span className="eyebrow">ПРОВЕРКА</span><h1>Введите код</h1><p>Код отправлен на <strong>{email}</strong>.</p>
    <form className="form-stack" onSubmit={handleSubmit(({ code }) => verify.mutate(code))}>
      <label className="field"><span>Одноразовый код</span><input className="otp-input" autoFocus inputMode="numeric" maxLength={6} placeholder="000000" {...register('code')} />{errors.code ? <small className="field-error">{errors.code.message}</small> : null}</label>
      {apiError ? <Alert>{apiError.message}{attemptsLeft !== undefined ? ` Осталось попыток: ${String(attemptsLeft)}.` : ''}{lockedUntil ? ` Блокировка до ${new Date(String(lockedUntil)).toLocaleString('ru-RU')}.` : ''}</Alert> : null}
      <Button type="submit" size="lg" loading={verify.isPending} disabled={apiError?.code === 'LOCKED'}>Войти</Button>
      <Button type="button" variant="ghost" onClick={() => navigate('/auth/login')}>Указать другую почту</Button>
    </form>
  </Card></section>;
}
