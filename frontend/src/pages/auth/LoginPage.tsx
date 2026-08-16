import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Mail } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { useLocation, useNavigate } from 'react-router-dom';
import { z } from 'zod';
import { authApi } from '../../api/endpoints';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { getErrorMessage } from '../../utils/errors';

const schema = z.object({
  email: z.email('Введите корректный email').refine((email) => /@(hse\.ru|edu\.hse\.ru|miem\.hse\.ru)$/i.test(email), 'Используйте корпоративную почту НИУ ВШЭ'),
});
type FormValues = z.infer<typeof schema>;

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const from = (location.state as { from?: string } | null)?.from;
  const { register, handleSubmit, formState: { errors } } = useForm<FormValues>({ resolver: zodResolver(schema) });
  const requestOtp = useMutation({
    mutationFn: (email: string) => authApi.requestOtp(email),
    onSuccess: (data, email) => {
      sessionStorage.setItem('miem3d_pending_email', email);
      sessionStorage.setItem('miem3d_otp_expires_in', String(data.expires_in));
      if (from) sessionStorage.setItem('miem3d_after_auth', from);
      else sessionStorage.removeItem('miem3d_after_auth');
      navigate('/auth/verify');
    },
  });

  return <section className="auth-page"><Card className="auth-card"><div className="auth-icon"><Mail /></div><span className="eyebrow">ВХОД</span><h1>Корпоративная почта</h1><p>Мы отправим одноразовый шестизначный код на адрес НИУ ВШЭ.</p>
    <form className="form-stack" onSubmit={handleSubmit(({ email }) => requestOtp.mutate(email.trim().toLowerCase()))}>
      <label className="field"><span>Email</span><input autoFocus type="email" placeholder="name@edu.hse.ru" {...register('email')} />{errors.email ? <small className="field-error">{errors.email.message}</small> : null}</label>
      {requestOtp.isError ? <Alert>{getErrorMessage(requestOtp.error)}</Alert> : null}
      <Button type="submit" size="lg" loading={requestOtp.isPending}>Получить код</Button>
    </form>
    <small className="muted">Допустимые домены: hse.ru, edu.hse.ru, miem.hse.ru.</small>
  </Card></section>;
}
