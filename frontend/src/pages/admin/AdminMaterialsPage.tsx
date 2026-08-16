import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Edit3, Plus, Power, X } from 'lucide-react';
import { useState } from 'react';
import { adminMaterialsApi } from '../../api/endpoints';
import { ErrorState } from '../../components/ErrorState';
import { LoadingState } from '../../components/LoadingState';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import type { AdminMaterial } from '../../types/api';
import { getErrorMessage } from '../../utils/errors';
import { getMaterialColor } from '../../utils/format';

function MaterialCard({ material }: { material: AdminMaterial }) {
  const client = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(material.name);
  const [description, setDescription] = useState(material.description || '');
  const [colorName, setColorName] = useState('');
  const invalidate = () => client.invalidateQueries({ queryKey: ['admin-materials'] });
  const update = useMutation({ mutationFn: (payload: Partial<Pick<AdminMaterial, 'name' | 'description' | 'is_active'>>) => adminMaterialsApi.update(material.id, payload), onSuccess: async () => { setEditing(false); await invalidate(); } });
  const addColor = useMutation({ mutationFn: () => adminMaterialsApi.addColor(material.id, { name: colorName.trim(), is_active: true }), onSuccess: async () => { setColorName(''); await invalidate(); } });
  const updateColor = useMutation({ mutationFn: ({ id, active }: { id: string; active: boolean }) => adminMaterialsApi.updateColor(material.id, id, { is_active: active }), onSuccess: invalidate });

  return <Card className={material.is_active ? 'catalog-admin-card' : 'catalog-admin-card catalog-admin-card--inactive'}><div className="catalog-admin-card__header"><div>{editing ? <div className="inline-edit"><input value={name} onChange={(event) => setName(event.target.value)} /><input value={description} onChange={(event) => setDescription(event.target.value)} /><Button size="sm" disabled={!name.trim()} loading={update.isPending} onClick={() => update.mutate({ name: name.trim(), description: description.trim() || null })}><Check size={15} /></Button><Button size="sm" variant="ghost" onClick={() => setEditing(false)}><X size={15} /></Button></div> : <><h3>{material.name}</h3><p>{material.description || 'Без описания'}</p></>}</div><div className="button-row"><Button size="sm" variant="ghost" onClick={() => setEditing(true)}><Edit3 size={15} /> Изменить</Button><Button size="sm" variant={material.is_active ? 'secondary' : 'primary'} loading={update.isPending} onClick={() => update.mutate({ is_active: !material.is_active })}><Power size={15} /> {material.is_active ? 'Отключить' : 'Включить'}</Button></div></div>
    <div className="admin-colors">{material.colors.map((color) => <button key={color.id} className={color.is_active ? 'admin-color' : 'admin-color admin-color--inactive'} onClick={() => updateColor.mutate({ id: color.id, active: !color.is_active })}><span style={{ background: getMaterialColor(color.name) }} />{color.name}<small>{color.is_active ? 'активен' : 'отключён'}</small></button>)}</div>
    <div className="inline-add"><input value={colorName} onChange={(event) => setColorName(event.target.value)} placeholder="Название нового цвета" /><Button size="sm" disabled={!colorName.trim()} loading={addColor.isPending} onClick={() => addColor.mutate()}><Plus size={16} /> Добавить цвет</Button></div>
    {update.isError || addColor.isError || updateColor.isError ? <Alert>{getErrorMessage(update.error || addColor.error || updateColor.error)}</Alert> : null}
  </Card>;
}

export function AdminMaterialsPage() {
  const client = useQueryClient();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const query = useQuery({ queryKey: ['admin-materials'], queryFn: adminMaterialsApi.list });
  const create = useMutation({ mutationFn: () => adminMaterialsApi.create({ name: name.trim(), description: description.trim() || undefined, is_active: true }), onSuccess: async () => { setName(''); setDescription(''); await client.invalidateQueries({ queryKey: ['admin-materials'] }); } });
  if (query.isPending) return <LoadingState />;
  if (query.isError) return <ErrorState message={getErrorMessage(query.error)} onRetry={() => void query.refetch()} />;
  return <div className="admin-section"><Card className="new-material-card"><h2>Новый материал</h2><div className="inline-add inline-add--wide"><input value={name} onChange={(event) => setName(event.target.value)} placeholder="Название, например PETG" /><input value={description} onChange={(event) => setDescription(event.target.value)} placeholder="Краткое описание" /><Button disabled={!name.trim()} loading={create.isPending} onClick={() => create.mutate()}><Plus size={17} /> Создать</Button></div>{create.isError ? <Alert>{getErrorMessage(create.error)}</Alert> : null}</Card><div className="catalog-admin-list">{query.data.map((material) => <MaterialCard material={material} key={material.id} />)}</div></div>;
}
