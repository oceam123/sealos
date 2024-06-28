import { NextApiRequest, NextApiResponse } from 'next';
import { jsonRes } from '@/services/backend/response';
import * as crypto from 'crypto';

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  try {
    const { name,password}=req.body

   if(name && password){
      const body={
        name:name,
        password:password
      }
      const infoStr=  btoa(JSON.stringify(body));
      console.log('infoStr',infoStr)
      res.redirect(`/ssoRLogin?infoStr=${infoStr}`)
      return;
    }
    return jsonRes(res, {
      message: 'Failed to get postLogin',
      code: 500
    });
  } catch (err) {
    console.log(err);
    return jsonRes(res, {
      message: 'Failed to get info',
      code: 500
    });
  }
}
