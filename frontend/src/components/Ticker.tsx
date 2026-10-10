import { fmtPt, fmtPtInt, serviceName } from '../lib/services'

interface TickerProps {
  totalTrips: number
  onTimePercent: number
  worstService: { code: string; avgDelay: number } | null
}

export function Ticker({ totalTrips, onTimePercent, worstService }: TickerProps) {
  const text = (
    <>
      &nbsp;{fmtPtInt(totalTrips)} VIAGENS SEGUIDAS ESTA SEMANA &nbsp;·&nbsp; <b>{fmtPt(onTimePercent)}%</b> PONTUAIS
      &nbsp;·&nbsp;
      {worstService && (
        <>
          {' '}
          PIOR SERVIÇO: {serviceName(worstService.code).toUpperCase()}, MÉDIA{' '}
          <b>+{fmtPt(worstService.avgDelay)} MIN</b> &nbsp;·&nbsp;
        </>
      )}{' '}
      DADOS RECOLHIDOS DA API DA CP A CADA 25 MIN &nbsp;·&nbsp; ESTE NÃO É UM SITE OFICIAL DA CP &nbsp;·&nbsp;
    </>
  )

  return (
    <div className="ticker" aria-hidden="true">
      <div className="ticker__track">
        {text}
        {text}
      </div>
    </div>
  )
}
