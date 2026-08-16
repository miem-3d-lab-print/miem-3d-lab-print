import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Save } from 'lucide-react';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { useLocation, useNavigate } from 'react-router-dom';
import { z } from 'zod';
import { profileApi } from '../api/endpoints';
import { useAuth } from '../auth/AuthContext';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { getErrorMessage } from '../utils/errors';

const schema = z.object({
  full_name: z.string().trim().min(2, 'Укажите ФИО'),
  telegram: z.string(),
  max: z.string(),
}).refine((value) => value.telegram.trim() || value.max.trim(), { message: 'Укажите Telegram или MAX', path: ['telegram'] });
type FormValues = z.infer<typeof schema>;

export function ProfilePage() {
  const { profile, refreshProfile } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const routeState = location.state as { profileRequired?: boolean; afterAuth?: string } | null;
  const required = Boolean(routeState?.profileRequired || (profile && (!profile.full_name?.trim() || (!profile.telegram?.trim() && !profile.max?.trim()))));
  const afterAuth = routeState?.afterAuth || sessionStorage.getItem('miem3d_after_auth');
  const { register, handleSubmit, reset, formState: { errors } } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { full_name: '', telegram: '', max: '' } });
  useEffect(() => {
    if (profile) reset({ full_name: profile.full_name || '', telegram: profile.telegram || '', max: profile.max || '' });
  }, [profile, reset]);
  const update = useMutation({
    mutationFn: (values: FormValues) => profileApi.update({ full_name: values.full_name.trim(), telegram: values.telegram.trim() || null, max: values.max.trim() || null }),
    onSuccess: async () => {
      await refreshProfile();
      if (required) {
        sessionStorage.removeItem('miem3d_after_auth');
        navigate(afterAuth || '/app/applications', { replace: true });
      }
    },
  });

  return <section className="page page--narrow"><div className="page-heading"><div><span className="eyebrow">АККАУНТ</span><h1>Мой профиль</h1><p>{required ? 'Заполните профиль перед первой заявкой.' : 'Актуальные контакты видны администраторам лаборатории.'}</p></div></div>
    <Card className="form-card"><form className="form-stack" onSubmit={handleSubmit((values) => update.mutate(values))}>
      <label className="field"><span>ФИО</span><input {...register('full_name')} />{errors.full_name ? <small className="field-error">{errors.full_name.message}</small> : null}</label>
      <label className="field"><span>Корпоративная почта</span><input value={profile?.email || ''} disabled /><small>Почта привязана к авторизации и не редактируется.</small></label>
      <div className="two-columns"><label className="field"><span>Telegram</span><input placeholder="@username" {...register('telegram')} />{errors.telegram ? <small className="field-error">{errors.telegram.message}</small> : null}</label><label className="field"><span>MAX</span><input placeholder="Ссылка или ник" {...register('max')} /></label></div>
      {update.isError ? <Alert>{getErrorMessage(update.error)}</Alert> : null}
      {update.isSuccess ? <Alert tone="success">Изменения сохранены.</Alert> : null}
      <div className="button-row"><Button type="submit" loading={update.isPending}><Save size={17} /> Сохранить</Button>{!required ? <Button type="button" variant="secondary" onClick={() => navigate(-1)}>Назад</Button> : null}</div>
    </form></Card>
  </section>;
}
