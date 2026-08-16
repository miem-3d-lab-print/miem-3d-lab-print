import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Navigate, useLocation, useNavigate } from 'react-router-dom';
import { profileApi } from '../api/endpoints';
import { useAuth } from '../auth/AuthContext';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { getErrorMessage } from '../utils/errors';

export function ConsentPage() {
  const [checked, setChecked] = useState(false);
  const { profile, refreshProfile } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const afterAuth = (location.state as { afterAuth?: string | null } | null)?.afterAuth || sessionStorage.getItem('miem3d_after_auth');
  const consent = useMutation({
    mutationFn: profileApi.consent,
    onSuccess: async () => {
      const updated = await refreshProfile();
      const incomplete = !updated.full_name || (!updated.telegram && !updated.max);
      if (incomplete) {
        navigate('/profile', { replace: true, state: { profileRequired: true, afterAuth } });
      } else {
        sessionStorage.removeItem('miem3d_after_auth');
        navigate(afterAuth || '/app/applications', { replace: true });
      }
    },
  });
  if (profile?.consent_given) return <Navigate to={afterAuth || '/app/applications'} replace />;
  return <section className="auth-page"><Card className="auth-card auth-card--wide"><span className="eyebrow">ПЕРВЫЙ ВХОД</span><h1>Согласие на обработку данных</h1><p>Для подачи и сопровождения заявки системе необходимо хранить ФИО, корпоративную почту, контакт и сведения о заказе.</p>
    <label className="consent-box"><input type="checkbox" checked={checked} onChange={(event) => setChecked(event.target.checked)} /><span>Я даю согласие на обработку персональных данных для работы сервиса 3D-печати.</span></label>
    {consent.isError ? <Alert>{getErrorMessage(consent.error)}</Alert> : null}
    <Button size="lg" disabled={!checked} loading={consent.isPending} onClick={() => consent.mutate()}>Подтвердить согласие</Button>
  </Card></section>;
}
