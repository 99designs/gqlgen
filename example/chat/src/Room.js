import React, { useState, useEffect, useRef } from 'react';
import styled from 'styled-components';
import gql from 'graphql-tag';
import { useQuery, useMutation } from '@apollo/react-hooks';

export const Chat = styled.div`
    padding: 4px;
    margin: 0 0 12px;
    max-width: 400px;
    max-height: 400px;
    border: 1px solid #ccc;
    background-color: #babdc6;
    overflow-x: hidden;
    overflow-y: scroll;
    position: relative;
    font: 14px verdana;
`;

export const ChatContainer = styled.div`
    &:after {
        content: "";
        display: table;
        clear: both;
    }
`;

export const Message = styled.div`
    color: #000;
    clear: both;
    line-height: 120%;
    padding: 8px;
    position: relative;
    margin: 8px 0;
    max-width: 85%;
    word-wrap: break-word;
    z-index: 1;

  &:after {
    position: absolute;
    content: "";
    width: 0;
    height: 0;
    border-style: solid;
  }

  span {
    display: inline-block;
    float: right;
    padding: 0 0 0 7px;
    position: relative;
    bottom: -4px;
  }

}`;

const MessageReceived = styled(Message)`
    background: #fff;
    border-radius: 0px 5px 5px 5px;
    float: left;

    &:after {
        border-width: 0px 10px 10px 0;
        border-color: transparent #fff transparent transparent;
        top: 0;
        left: -4px;
    }

    span {
        display: block;
        color: #bbb;
        font-size: 10px;
    }
`;

const MessageMine = styled(Message)`
    background: #e1ffc7;
    border-radius: 5px 0px 5px 5px;
    float: right;

    &:after {
        border-width: 0px 0 10px 10px;
        border-color: transparent transparent transparent #e1ffc7;
        top: 0;
        right: -4px;
    }
`;

export const Room = ({ channel, name }) => {
    const messagesEndRef = useRef(null)
    const [ text, setText ] = useState('');

    const [ addMessage ] = useMutation(MUTATION, {
        onCompleted: () => {
            setText('');
        }
    });

    const { loading, error, data, subscribeToMore } = useQuery(QUERY, {
        variables: {
            channel
        },
    });

    // subscribe to more messages
    useEffect(() => {
        const subscription = subscribeToMore({
            document: SUBSCRIPTION,
            variables: {
                channel,
            },
            updateQuery: (prev, { subscriptionData }) => {

                if (!subscriptionData.data) {
                    return prev;
                }
                const newMessage = subscriptionData.data.messageAdded;

                if (prev.room.messages.find((msg) => msg.id === newMessage.id)) {
                    return prev
                }

                return Object.assign({}, prev, {
                    room: Object.assign({}, prev.room, {
                        messages: [...prev.room.messages, newMessage],
                    })
                });
            },
        });


        return () => subscription();

    }, [subscribeToMore, channel]);

    // auto scroll down
    useEffect(() => {
        messagesEndRef && messagesEndRef.current && messagesEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }, [messagesEndRef, data]);

    if (loading) {
        return <div>loading</div>
    }

    if (error) {
        return <div>error</div>
    }

    return (<>
        <Chat>
            <ChatContainer>
                {data.room.messages.map((msg) =>
                    msg.createdBy === name ? <MessageMine key={msg.id}>
                        {msg.text}
                    </MessageMine> : <MessageReceived key={msg.id}>
                        <span>{msg.createdBy}</span>
                        {msg.text}
                    </MessageReceived>
                )}
            </ChatContainer>
            <div ref={messagesEndRef} />
        </Chat>

        <input value={text} onChange={(e) => setText(e.target.value)} />

        <p>
            <button
                onClick={() => addMessage({
                    variables: {
                        text: text,
                        channel: channel,
                        name: name,
                    }
                    })
                } >
                send
            </button>
        </p>
    </>);

}

const SUBSCRIPTION = gql`
    subscription MoreMessages($channel: String!) {
        messageAdded(roomName:$channel) {
            id
            text
            createdBy
        }
    }
`;

const QUERY = gql`
    query Room($channel: String!) {
        room(name: $channel) {
            messages { id text createdBy }
        }
    }
`;

const MUTATION = gql`
    mutation sendMessage($text: String!, $channel: String!, $name: String!) {
        post(text:$text, roomName:$channel, username:$name) { id }
    }
`;
