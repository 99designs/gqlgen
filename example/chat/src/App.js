import React, { useState } from 'react';
import styled from 'styled-components';
import { Room } from './Room';

const Input = styled.div`
    padding: 4px;
    margin: 0 0 4px;

    input {
        border: 1px solid #ccc;
        padding: 2px;
        font-size: 14px;
    }
`;

export const App = () => {
    const [name, setName] = useState('tester');
    const [channel, setChannel] = useState('#gophers');

    return (
        <>
            <Input>
                name: <input value={name} onChange={(e) => setName(e.target.value)} />
            </Input>
            <Input>
                channel: <input value={channel} onChange={(e) => setChannel(e.target.value)} />
            </Input>

            <Room channel={channel} name={name} />
        </>
    );

};
