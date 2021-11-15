import styled from 'styled-components';

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

const ChatLine = styled.div`
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

export const MessageReceived = styled(ChatLine)`
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

export const Message = styled(ChatLine)`
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
