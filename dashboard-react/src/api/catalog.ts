import { adminGet, adminSend } from './client'
import { apiPaths } from './paths'
import type { Channel, ChannelBalanceInput, ChannelCreateInput, ChannelModel, ChannelModelInput, ChannelModelUpdate, ChannelStatusInput, ChannelTestItem, CatalogModel, Deleted, Health, ListResponse, Pricing, PricingCreateInput, RemoteModelsResult, ChannelUpdateInput, DeletePricingInput } from '../types/api'


export const listChannels = () => adminGet<ListResponse<Channel>>(apiPaths.channels(), { page: 1, page_size: 100 })
export const listChannelHealth = () => adminGet<ListResponse<Health>>(apiPaths.channelHealthList())
export const createChannel = (input: ChannelCreateInput) => adminSend<Channel, ChannelCreateInput>('POST', apiPaths.channels(), input)
export const updateChannel = (id: number, input: ChannelUpdateInput) => adminSend<Channel, ChannelUpdateInput>('PUT', apiPaths.channel(id), input)
export const deleteChannel = (id: number) => adminSend<Deleted>('DELETE', apiPaths.channel(id))
export const updateChannelStatus = (id: number, input: ChannelStatusInput) => adminSend<Channel, ChannelStatusInput>('PUT', apiPaths.channelStatus(id), input)
export const updateChannelBalance = (id: number, input: ChannelBalanceInput) => adminSend<Channel, ChannelBalanceInput>('PUT', apiPaths.channelBalance(id), input)
export const resetChannelHealth = (id: number) => adminSend<Deleted>('POST', apiPaths.channelHealthReset(id))
export const listChannelModels = (channelId: number) => adminGet<ListResponse<ChannelModel>>(apiPaths.channelModels(channelId))
export const createChannelModel = (channelId: number, input: ChannelModelInput) => adminSend<ChannelModel, ChannelModelInput>('POST', apiPaths.channelModels(channelId), input)
export const updateChannelModel = (channelId: number, modelId: number, input: ChannelModelUpdate) => adminSend<ChannelModel, ChannelModelUpdate>('PUT', apiPaths.channelModel(channelId, modelId), input)
export const deleteChannelModel = (channelId: number, modelId: number) => adminSend<Deleted>('DELETE', apiPaths.channelModel(channelId, modelId))
export const loadRemoteModels = (channelId: number) => adminSend<RemoteModelsResult>('POST', apiPaths.channelRemoteModels(channelId))
export const testChannel = (channelId: number, checkAll: boolean) => adminSend<{ list: ChannelTestItem[] }, { check_all: boolean }>('POST', apiPaths.channelTest(channelId), { check_all: checkAll })
export const listModels = (status = 1) => adminGet<ListResponse<CatalogModel>>(apiPaths.models(), { status })
export const listPricing = () => adminGet<ListResponse<Pricing>>(apiPaths.pricing(), { page: 1, page_size: 100 })
export const createPricing = (input: PricingCreateInput) => adminSend<Pricing, PricingCreateInput>('POST', apiPaths.pricing(), input)
export const deletePricing = (input: DeletePricingInput) => adminSend<Deleted, DeletePricingInput>('DELETE', apiPaths.pricing(), input)
