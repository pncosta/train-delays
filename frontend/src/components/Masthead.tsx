export function Masthead() {
  return (
    <header className="masthead">
      <p className="eyebrow upper">a seguir aos comboios, ao vivo</p>
      <h1 className="title">
        ATRAS<em>O</em>S<span style={{ color: 'var(--ink-faint)' }}>.</span>
      </h1>
      <p className="tagline">
        Um rastreador independente da pontualidade dos comboios em Portugal —{' '}
        <b>a CP não publica estes números</b>, por isso recolhemo-los nós, estação a estação, todos os dias.
      </p>
    </header>
  )
}
