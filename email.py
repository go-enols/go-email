# -*- coding: utf-8 -*-
import email as email_reader
import random
import ssl
from email.header import decode_header
from enum import Enum
import requests
import imaplib

def get_access_token_from_refresh_token(refresh_token, client_id):
    headers = {
        'Host': 'login.microsoftonline.com',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36',
        'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8',
    }
    data = {
        "client_id": client_id,
        "refresh_token": refresh_token,
        "grant_type": "refresh_token"
    }
    rr = requests.post("https://login.microsoftonline.com/common/oauth2/v2.0/token", headers=headers, data=data)

    if rr.json().get("error") is None:
        return {"code": 0, "access_token": rr.json()["access_token"], "refresh_token": rr.json()["refresh_token"]}
    if rr.json().get("error_description").find("User account is found to be in service abuse mode") != -1:
        return {"code": 1, "message": "account was blocked or wrong username,password,refresh_token,client_id"}
    return {"code": 1, "message": "get access token is wrong"}


def imap_authenticate_with_oauth2(username, access_token):
        auth_string = f"user={username}\1auth=Bearer {access_token}\1\1"
        mail = imaplib.IMAP4_SSL("outlook.office365.com")
        mail.authenticate("XOAUTH2", lambda x: auth_string)
        return mail
def read_mail(email, access_token):
    mail = imap_authenticate_with_oauth2(email, access_token)

    mail.list()
    mail.select("inbox")
    status, messages = mail.search(None, 'ALL')

    messages = messages[0].split()
    if not messages:
        print("No emails found in inbox")
        return

    # 获取最后一封邮件（最新的邮件）
    latest_email_id = messages[-1]
    
    # 只获取最新的一封邮件
    status, msg_data = mail.fetch(latest_email_id, '(RFC822)')
    email_body = msg_data[0][1]
    message = email_reader.message_from_bytes(email_body)
    
    # Get email subject
    subject = decode_header(message["Subject"])[0][0]
    if isinstance(subject, bytes):
        subject = subject.decode()
    print(f"Subject: {subject}")
    
    # Get sender
    from_ = decode_header(message.get("From", ""))[0][0]
    if isinstance(from_, bytes):
        from_ = from_.decode()
    print(f"From: {from_}")
    
    # Get date
    date_ = message.get("Date", "")
    print(f"Date: {date_}")
    
    # Get body
    if message.is_multipart():
        # Handle multipart messages
        for part in message.walk():
            if part.get_content_type() == "text/plain":
                body = part.get_payload(decode=True).decode()
                print(f"Body: {body[:200]}...")  # Print first 200 chars
                break
    else:
        # Handle plain text emails
        body = message.get_payload(decode=True).decode()
        print(f"Body: {body}")  # Print first 200 chars
        
    print("-" * 50)  # Separator
    print("Done reading latest email")
    print("done")

#oauth2认证至少需要email, clientid, refresh_token
#下面的demo通过email,clientid,refresh_token获取access_token, 然后使用access_token读取邮件列表, 更多功能自行开发
#卡密(格式:email----密码----clientid----refresh_token----access_token)
#wendeyvenys@hotmail.com----vp15Rz60----8b4ba9dd-3ea5-4e5f-86f1-ddba2230dcf2----M.C525_BAY.0.U.-CvxMhLe1rlmKZ3H6fEdCrNRAggUCP5Z55X1C3lwChG0YP7CbpvsWBNTR778D8VEcZ2YPv8iUGUCWxx*f0j4SIjSz3HX!4pFIvFD1OR6udxZEiv*6E!2sqwU4qoYk410ie31TxjhVaUoXbZ5xXCHYxvYzJjPEZ1PeibIxBau4N!Duie73iVM!vGY81tJuFLPTFahzdB6Eu9CftyxqloI7TO*elPnuIIr0NJ*4g1vcQV0qiJuUUOKGDBFIaRUaUtbahmS66O*ZqOub5HoiZ*zZLHeZ!vAL9usCz3TOsMI82KxCK50dgSv87dbn1tryQ!VrKWpZcoPlP6YgjWFelADWSJMBAvnUOTWkBjNclHO35WnqeMh!OdtWqmP1HktZWCPEytOBU8RynGl7Z2j7ct!Q7tqUoOmiczZ!y1LGK3deUFPWA9IHdtJYxl2yEGBYkbQbDA$$----EwBIA+l3BAAUcDnR9grBJokeAHaUV8R3+rVHX+IAASbeXJLPrO1Qjwsp0nwbuisWvhOIRrVtIblqjjms95h3mqKzHZ0KKlyhW0ZQQXzqjgfZg2tJ1I10MF4/2cSZ0v57vZTuAT/i0ffutChMIDvH+KfLu81Iy12Urc3c3oe3FNe+tRCa56xkWdooZsXwXhTRkdT7gEnSaIXYoDCknCgNojvxIqhbS0dfWkHvukPXk8xEPN3EQdv0aLbsqx9sUFE6hWp7cR3xTLytlIIVqyruOI9nv1sh36F0l3lCWdp5WGVfbh+vYK4RBnXvsPtQgEJ9bxu54haIy9kwOHOqK8CliRQ6R2gVpDoyZNQT1evoTykDCg+/sNnoqkWS6P9GOl4QZgAAEPSRDzdxDfET2MevAKNvugoQArGRJZq1c2cm9bxcb+ec5NrQ9cMV64TgFA7ZX+XwU1xbLkgYpOtWeQjksOCZ2XCFzrN+ibw2hN6LNnctWJBXOaYtVWzu3xwdy0+oqRRbEfsTayXMwHprGkUjXtY970JD1Mb0iOOpXrb4V8YTqAaA/IDYpYtWF5B2+ZLwYrWqvqZFuSBvi7852TgUMxr2z7UA5qDCwN40q1r+6tolH7Jde9P2858Bn3lTAybNtZ+4cqiu5GXnAC4QDDQ8lHiCaNLFGalzlRcGA4PYzaRtzG/88cFhurvBqCCscgom52T5PCUttoMpH/b3hJ9QB7YBMkvvJNMRAUZVswAiEt5GchIDaldAYuGLF1NbfeEtNiDmvkEvLx//aIlySwJFB43B/VuqzY4D2aFSWjXoqg0LD5BGICJb345y642+as+1i/uwtwU6hR5tTgkxVA999qFAUR2PcREdEjGaWWqSdm33WTCcY+8yiKfCUNJ6jRa+lzMTyZxnuto6TuLSgKDhG9hkhfx1hoqyljpc/rr275XmYQMxlDsBDSQSFzjhrtf/5SBoeKMRPenKs0Ptkbv5qin7k/uS/P2zoMAcHdg69qZS6zwWIuV1xwEWh8OjymSZJMDKZ+yea09rWWE5kGQvLTs4RwJ/JJXXoOYFjHK3wIZjhN29druDTnFlCBLn/iMy0gt7XKgemX5i8kZNXBDXuV5xrA+8QEcC
def example():
    data = "ugqya4974519@hotmail.com----izfosfm6092----dbc8e03a-b00c-46bd-ae65-b683e7707cb0----M.C531_BAY.0.U.-CmJENJzwQ8IctY3hiAbn6HM1shWRbBY4XdvyHYVUQqY4NiJ2TU*fObvI!FD4lJwMY*Gm8yMJlUoiX4IVaBfHTDMvDRezioTNxyBOSqJ!vQQZSc82r8BssZpXoZu83ll51tZ6s1VOx4xBV6gTJdL5qFPLTQ37!C2m3isAf6FPUh!1vFAqHM5rYiiO9jMDVTFUq!7GbBqQ7k9*qFyTXR3UpEefZ*pTxg1alXK5d9HMWRCExhW2J*soQilrjIphGMOvsnEy3hElFP9rOtyHKpHgj0nA2lcnHzA31N113Q!ASXr85guzJpV84Vkp9keoUBfB5m9KbFJu4uiR0AQzDPyHwtQOd9jNEcvqX5IYCMYZF0mwpciEhcj6JJKnooIDGQf3wavybWzz50Wd*SC9PIqRQ04$"
    data = data.split("----")
    
    email=data[0]
    client_id=data[2]
    refresh_token= data[3]
    token=get_access_token_from_refresh_token(refresh_token, client_id)
    read_mail(email, token["access_token"])

#调用示例
example()