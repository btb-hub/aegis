import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../i18n';
import { IncidentDetail } from './IncidentDetail';
import type { Incident } from '../../lib/incidentTypes';
const incident: Incident = {id:'incident-1',teamId:'team',status:'resolved',severity:'warning',title:'CPU',fingerprint:'fp',createdAt:'2026-10-05T00:00:00Z',alerts:[],timeline:[]};
it('adds a comment to a resolved incident and preserves line breaks',async()=>{
 const add=vi.fn().mockResolvedValue(undefined);
 render(<I18nextProvider i18n={i18n}><IncidentDetail incident={incident} teams={[]} canBounce={false} onAcknowledge={()=>{}} onResolve={()=>{}} onHandoff={()=>{}} onBounce={()=>{}} onComment={add}/></I18nextProvider>);
 fireEvent.change(screen.getByLabelText('Comment'),{target:{value:'Investigating\nCache restored'}});
 fireEvent.click(screen.getByRole('button',{name:'Add comment'}));
 await waitFor(()=>expect(add).toHaveBeenCalledWith('incident-1','Investigating\nCache restored'));
});
describe('resolution note',()=>{it('opens a dialog before resolving',()=>{
 const resolve=vi.fn();render(<I18nextProvider i18n={i18n}><IncidentDetail incident={{...incident,status:'open'}} teams={[]} canBounce={false} onAcknowledge={()=>{}} onResolve={resolve} onHandoff={()=>{}} onBounce={()=>{}}/></I18nextProvider>);
 fireEvent.click(screen.getByRole('button',{name:'Resolve'}));expect(resolve).not.toHaveBeenCalled();expect(screen.getByRole('dialog')).toBeInTheDocument();
})});

it('submits a resolution note and keeps it when saving fails',async()=> {
 const resolve=vi.fn().mockResolvedValue(false);
 render(<I18nextProvider i18n={i18n}><IncidentDetail incident={{...incident,status:'open'}} teams={[]} canBounce={false} onAcknowledge={()=>{}} onResolve={resolve} onHandoff={()=>{}} onBounce={()=>{}}/></I18nextProvider>);
 fireEvent.click(screen.getByRole('button',{name:'Resolve'}));
 fireEvent.change(screen.getByLabelText('Resolution comment (optional)'),{target:{value:'Database restored\nVerified'}});
 fireEvent.click(screen.getAllByRole('button',{name:'Resolve'})[1]);
 await waitFor(()=>expect(resolve).toHaveBeenCalledWith('incident-1','Database restored\nVerified'));
 await screen.findByText('Comment was not saved. Check the connection and try again.');
 expect(screen.getByRole('dialog')).toBeInTheDocument();
 resolve.mockResolvedValue(true);fireEvent.click(screen.getAllByRole('button',{name:'Resolve'})[1]);await waitFor(()=>expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
});
it('hides mutations for viewers and renders comments as plain text',()=> {
 render(<I18nextProvider i18n={i18n}><IncidentDetail incident={{...incident,timeline:[{id:'event',kind:'comment_added',createdAt:'2026-10-05T00:00:00Z',payload:{body:'<script>bad</script>\nLine',author_name:'Engineer'}}]}} teams={[]} canBounce={false} canMutate={false} onAcknowledge={()=>{}} onResolve={()=>{}} onHandoff={()=>{}} onBounce={()=>{}} onComment={vi.fn()}/></I18nextProvider>);
 expect(screen.queryByRole('button',{name:'Add comment'})).not.toBeInTheDocument();expect(screen.getByText(/<script>bad/)).toHaveClass('whitespace-pre-wrap');expect(document.querySelector('script')).toBeNull();
});
