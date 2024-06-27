import type { NextPage } from 'next';
import { passwordExistRequest, passwordLoginRequest, UserInfo } from '@/api/auth';
import useSessionStore from '@/stores/session';
import { jsonRes } from '@/services/backend/response';
import { getInviterId, sessionConfig } from '@/utils/sessionConfig';
import { jwtDecode } from 'jwt-decode';
import { AccessTokenPayload } from '@/types/token';
import { useRouter } from 'next/router';
import { useEffect } from 'react';
import { useTranslation } from 'next-i18next';
import { Flex, Spinner } from '@chakra-ui/react';



const Callback: NextPage = () => {
 
const { t } = useTranslation();
const router = useRouter();
const setToken = useSessionStore((s) => s.setToken);
const setSession = useSessionStore((s) => s.setSession);

  useEffect( () => {
    const fetchData = async () => {
       const name =router.query.name as string
       const password=router.query.password as string
       if (name &&password) {
         try {
           const inviterId = getInviterId();
           const result = await passwordExistRequest({ user: name });
   
           if (result?.code === 200) {
             const result = await passwordLoginRequest({
               user: name,
               password: password,
               inviterId
             });
             if (!!result?.data) {
              console.log('sessionConfig')
               await sessionConfig(result.data);
             }
             console.log('sessionConfig success')
             await router.replace('/');
           } else if (result?.code === 201) {
               const regionResult = await passwordLoginRequest({
                 user: name,
                 password: password,
                 inviterId
               });
               if (!!regionResult?.data) {
                 setToken(regionResult.data.token);
                 const infoData = await UserInfo();
                 const payload = jwtDecode<AccessTokenPayload>(regionResult.data.token);
                 setSession({
                   token: regionResult.data.token,
                   user: {
                     k8s_username: payload.userCrName,
                     name: infoData.data?.info.nickname || '',
                     avatar: infoData.data?.info.avatarUri || '',
                     nsid: payload.workspaceId,
                     ns_uid: payload.workspaceUid,
                     userCrUid: payload.userCrUid,
                     userId: payload.userId,
                     userUid: payload.userUid
                   },
                   // @ts-ignore
                   kubeconfig: result.data.kubeconfig
                 });
                 await router.replace('/');
               }
             
           }
         } catch (error: any) {
          await router.replace('/signin');
          console.log('error')

         } 
       }
       
    };
   
    fetchData();
  }, [router]);
  return (
    <Flex w={'full'} h={'full'} justify={'center'} align={'center'}>
      <Spinner size="xl" />
    </Flex>
  );
};
export default Callback;
