import { useQuery } from '@tanstack/react-query';
import { ArrowRight, FileUp, Layers3, PackageCheck, Printer } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { materialsApi } from '../api/endpoints';
import { useAuth } from '../auth/AuthContext';
import { Button } from '../components/ui/Button';
import { getMaterialColor } from '../utils/format';

const fallbackMaterials = [
  { id: 'pla', name: 'PLA', description: 'Жёсткий и экологичный пластик', colors: [{ id: '1', name: 'Красный' }, { id: '2', name: 'Белый' }, { id: '3', name: 'Чёрный' }] },
  { id: 'abs', name: 'ABS', description: 'Прочный и ударостойкий материал', colors: [{ id: '4', name: 'Чёрный' }] },
  { id: 'petg', name: 'PETG', description: 'Хорошая межслойная адгезия', colors: [{ id: '5', name: 'Синий' }, { id: '6', name: 'Белый' }] },
  { id: 'tpu', name: 'TPU', description: 'Гибкий и эластичный материал', colors: [{ id: '7', name: 'Чёрный' }] },
];

export function HomePage() {
  const navigate = useNavigate();
  const { profile, isAuthenticated } = useAuth();
  const materials = useQuery({
    queryKey: ['materials'], queryFn: materialsApi.list,
    enabled: Boolean(isAuthenticated && profile?.consent_given),
  });
  const cards = materials.data ?? fallbackMaterials;
  const start = () => navigate(isAuthenticated ? '/app/applications/new' : '/auth/login', { state: { from: '/app/applications/new' } });

  return (
    <>
      <section className="hero-section">
        <div className="hero-copy">
          <span className="eyebrow">СЕРВИС 3D-ПЕЧАТИ</span>
          <h1>Подайте заявку на печать модели за пару минут</h1>
          <p>Загрузите файл, выберите материал и цвет — остальное сделает лаборатория. Статус заявки всегда виден в личном кабинете.</p>
          <div className="button-row">
            <Button size="lg" onClick={start}>Оформить заявку <ArrowRight size={18} /></Button>
            <Button size="lg" variant="secondary" onClick={() => document.getElementById('how')?.scrollIntoView({ behavior: 'smooth' })}>Как это работает</Button>
          </div>
        </div>
        <div className="hero-image"><img src="/images/hero.webp?v=20260816" alt="Интерфейс сервиса 3D-печати МИЭМ" /></div>
      </section>

      <section className="home-section home-section--white" id="how">
        <div className="section-heading"><div><span className="eyebrow">ПРОЦЕСС</span><h2>Как это работает</h2></div></div>
        <div className="steps-grid">
          {[
            [FileUp, 'Заявка', 'Заполните форму и загрузите STL, STEP, 3MF или ZIP-архив.'],
            [Layers3, 'Материал', 'Выберите доступный материал и, при необходимости, цвет.'],
            [Printer, 'Печать', 'Следите за изменением статуса в личном кабинете.'],
            [PackageCheck, 'Выдача', 'Получите готовую модель в аудитории 211.'],
          ].map(([Icon, title, text], index) => {
            const StepIcon = Icon as typeof FileUp;
            return <article className="step-card" key={String(title)}><span className="step-number">{index + 1}</span><StepIcon size={24} /><h3>{String(title)}</h3><p>{String(text)}</p></article>;
          })}
        </div>
      </section>

      <section className="home-section">
        <div className="section-heading"><div><span className="eyebrow">КАТАЛОГ</span><h2>Материалы и цвета</h2></div>{isAuthenticated ? <Button variant="ghost" onClick={() => navigate('/app/applications/new')}>Выбрать в заявке <ArrowRight size={17} /></Button> : null}</div>
        <div className="materials-grid">
          {cards.map((material) => <article className="material-card" key={material.id}>
            <div className="material-card__top"><h3>{material.name}</h3><span>{material.colors.length} цв.</span></div>
            <p>{material.description || 'Описание материала уточняется'}</p>
            <div className="color-dots">{material.colors.map((color) => <span key={color.id} title={color.name} style={{ background: getMaterialColor(color.name) }} />)}</div>
          </article>)}
        </div>
      </section>

      <section className="contact-band">
        <div><span className="eyebrow">КОНТАКТЫ</span><h2>Лаборатория 3D-визуализации</h2><p>Таллинская ул., 34, аудитория 211 · руководитель — Коваленко А. А.</p><p>aakovalenko@hse.ru · 8-495-772-95-90, доб. 15189/15863</p></div>
      </section>
    </>
  );
}
